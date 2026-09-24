package update

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/komari-monitor/komari-agent/dnsresolver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

var (
	CurrentVersion = "dev"
	Repo           = "berry-shake/komari-agent"
	maintenanceTag = regexp.MustCompile(`^1\.2\.13-fork\.[1-9][0-9]*$`)
)

const containerMarkerPath = "/.komari-agent-container"

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	HTMLURL     string        `json:"html_url"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}
type githubAsset struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Size               int    `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func parseVersion(ver string) (semver.Version, error) {
	return semver.ParseTolerant(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(ver), "v"), "V"))
}
func needUpdate(current, latest semver.Version) bool { return latest.Compare(current) > 0 }
func DoUpdateWorks() {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		if err := CheckAndUpdate(); err != nil {
			log.Println("Update check:", err)
		}
	}
}

// Only this maintenance line is eligible. Retained newer baselines are rollback
// artifacts and must never pull a single-database deployment forward again.
func selectMaintenanceRelease(releases []githubRelease) *githubRelease {
	var best *githubRelease
	for i := range releases {
		r := &releases[i]
		if r.Draft || r.Prerelease || !maintenanceTag.MatchString(r.TagName) {
			continue
		}
		version, err := parseVersion(r.TagName)
		if err != nil {
			continue
		}
		if best == nil {
			best = r
			continue
		}
		previous, _ := parseVersion(best.TagName)
		if needUpdate(previous, version) {
			best = r
		}
	}
	return best
}
func releaseForPlatform(r *githubRelease, goos, goarch string) (*selfupdate.Release, error) {
	name := "komari-agent-" + goos + "-" + goarch
	if goos == "windows" {
		name += ".exe"
	}
	var asset, validation *githubAsset
	for i := range r.Assets {
		a := &r.Assets[i]
		if a.Name == name {
			asset = a
		}
		if a.Name == name+".sha256" {
			validation = a
		}
	}
	if asset == nil || validation == nil {
		return nil, fmt.Errorf("release %s lacks binary or checksum for %s", r.TagName, name)
	}
	version, err := parseVersion(r.TagName)
	if err != nil {
		return nil, err
	}
	ownerRepo := strings.SplitN(Repo, "/", 2)
	if len(ownerRepo) != 2 {
		return nil, fmt.Errorf("invalid update repository")
	}
	return &selfupdate.Release{Version: version, AssetURL: asset.BrowserDownloadURL, AssetByteSize: asset.Size, AssetID: asset.ID, ValidationAssetID: validation.ID, URL: r.HTMLURL, ReleaseNotes: r.Body, Name: r.Name, PublishedAt: &r.PublishedAt, RepoOwner: ownerRepo[0], RepoName: ownerRepo[1]}, nil
}
func CheckAndUpdate() error {
	if _, err := os.Stat(containerMarkerPath); err == nil {
		log.Println("Container agent: update the berry-shake image instead of the running binary.")
		return nil
	}
	if CurrentVersion == "dev" {
		log.Println("Development build: automatic update is disabled.")
		return nil
	}
	if !maintenanceTag.MatchString(CurrentVersion) {
		return fmt.Errorf("version %q is outside the 1.2.13 fork maintenance line", CurrentVersion)
	}
	current, err := parseVersion(CurrentVersion)
	if err != nil {
		return err
	}
	log.Println("Checking update from", Repo, "(1.2.13 fork maintenance line)")
	http.DefaultClient = dnsresolver.GetHTTPClient(60 * time.Second)
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/"+Repo+"/releases?per_page=100", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("release API returned HTTP %d", resp.StatusCode)
	}
	var releases []githubRelease
	if err = json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&releases); err != nil {
		return err
	}
	selected := selectMaintenanceRelease(releases)
	if selected == nil {
		log.Println("No published release in the 1.2.13 fork maintenance line")
		return nil
	}
	latest, err := releaseForPlatform(selected, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	if !needUpdate(current, latest.Version) {
		log.Println("Current version is the latest:", CurrentVersion)
		return nil
	}
	updater, err := selfupdate.NewUpdater(selfupdate.Config{Validator: checksumValidator{}})
	if err != nil {
		return err
	}
	binary, err := os.Executable()
	if err != nil {
		return err
	}
	binary, err = filepath.EvalSymlinks(binary)
	if err != nil {
		return err
	}
	if err = updater.UpdateTo(latest, binary); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	log.Println("Successfully updated to version", latest.Version)
	os.Exit(42)
	return nil
}
