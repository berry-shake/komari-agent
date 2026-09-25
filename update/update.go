package update

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/blang/semver"
	goupdate "github.com/inconshreveable/go-update"
	"github.com/komari-monitor/komari-agent/dnsresolver"
	"github.com/komari-monitor/komari-agent/utils"
)

var (
	CurrentVersion = "dev"
	Repo           = "berry-shake/komari-agent"
	numericTag     = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	minimumVersion = semver.MustParse("1.2.4")
)

type platformRelease struct {
	Version           semver.Version
	AssetURL          string
	ChecksumURL       string
	AssetByteSize     int
	ValidationAssetID int64
	RepoOwner         string
	RepoName          string
}

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

// Version numbers belong to our distribution, independently of the upstream
// Agent baseline. Legacy fork tags are never considered upgrade candidates.
func isDistributionVersion(tag string) bool {
	if !numericTag.MatchString(tag) {
		return false
	}
	version, err := parseVersion(tag)
	return err == nil && version.Compare(minimumVersion) >= 0
}

func selectDistributionRelease(releases []githubRelease) *githubRelease {
	var best *githubRelease
	for i := range releases {
		r := &releases[i]
		if r.Draft || r.Prerelease || !isDistributionVersion(r.TagName) {
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
func releaseForPlatform(r *githubRelease, goos, goarch string) (*platformRelease, error) {
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
	return &platformRelease{Version: version, AssetURL: asset.BrowserDownloadURL, ChecksumURL: validation.BrowserDownloadURL, AssetByteSize: asset.Size, ValidationAssetID: validation.ID, RepoOwner: ownerRepo[0], RepoName: ownerRepo[1]}, nil
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
	if !isDistributionVersion(CurrentVersion) {
		return fmt.Errorf("version %q is outside the numeric distribution; install version 1.2.4 or later explicitly", CurrentVersion)
	}
	current, err := parseVersion(CurrentVersion)
	if err != nil {
		return err
	}
	log.Println("Checking update from", Repo, "(numeric distribution)")
	client := newUpdateClient()
	defer client.CloseIdleConnections()
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/"+Repo+"/releases?per_page=100", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
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
	selected := selectDistributionRelease(releases)
	if selected == nil {
		log.Println("No published numeric distribution release")
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
	binary, err := os.Executable()
	if err != nil {
		return err
	}
	binary, err = filepath.EvalSymlinks(binary)
	if err != nil {
		return err
	}
	if err = installRelease(client, latest, binary); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	log.Println("Successfully updated to version", latest.Version)
	os.Exit(42)
	return nil
}

const maxUpdateBytes int64 = 128 << 20

func trustedUpdateURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || (u.Port() != "" && u.Port() != "443") {
		return false
	}
	switch u.Hostname() {
	case "github.com", "api.github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com":
		return true
	default:
		return false
	}
}

func newUpdateClient() *http.Client {
	client := dnsresolver.NewSecureHTTPClient(60 * time.Second)
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 || !trustedUpdateURL(req.URL.String()) {
			return fmt.Errorf("untrusted update redirect")
		}
		return nil
	}
	return client
}

func downloadUpdateAsset(client *http.Client, rawURL string, limit int64) ([]byte, error) {
	if !trustedUpdateURL(rawURL) {
		return nil, fmt.Errorf("untrusted update asset URL")
	}
	response, err := client.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update download returned HTTP %d", response.StatusCode)
	}
	if response.ContentLength > limit {
		return nil, fmt.Errorf("update asset exceeds size limit")
	}
	return utils.ReadBounded(response.Body, limit)
}

// Only raw binaries from our numeric distribution are supported. Avoid archive
// decoders and their additional attack surface in the generic self-update SDK.
func installRelease(client *http.Client, release *platformRelease, binaryPath string) error {
	if release.AssetByteSize <= 0 || int64(release.AssetByteSize) > maxUpdateBytes {
		return fmt.Errorf("invalid update asset size")
	}
	checksum, err := downloadUpdateAsset(client, release.ChecksumURL, 16<<10)
	if err != nil {
		return err
	}
	binary, err := downloadUpdateAsset(client, release.AssetURL, maxUpdateBytes)
	if err != nil {
		return err
	}
	if len(binary) != release.AssetByteSize {
		return fmt.Errorf("update asset size mismatch")
	}
	if err := (checksumValidator{}).Validate(binary, checksum); err != nil {
		return err
	}
	return goupdate.Apply(bytes.NewReader(binary), goupdate.Options{TargetPath: binaryPath})
}
