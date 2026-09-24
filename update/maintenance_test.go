package update

import "testing"

func TestMaintenanceSelectionKeepsSingleDatabaseLine(t *testing.T) {
	releases := []githubRelease{{TagName: "1.2.60-fork.99"}, {TagName: "1.2.13-fork.2"}, {TagName: "1.2.13-fork.10"}, {TagName: "1.2.13-fork.100", Draft: true}, {TagName: "1.2.13-fork.101", Prerelease: true}, {TagName: "1.2.13"}, {TagName: "1.2.13-fork.01"}}
	if got := selectMaintenanceRelease(releases); got == nil || got.TagName != "1.2.13-fork.10" {
		t.Fatalf("selected %+v", got)
	}
	if got := selectMaintenanceRelease(releases[:1]); got != nil {
		t.Fatalf("accepted other baseline: %+v", got)
	}
}
func TestPlatformReleaseRequiresMatchingChecksum(t *testing.T) {
	r := &githubRelease{TagName: "1.2.13-fork.1", Assets: []githubAsset{{ID: 1, Name: "komari-agent-linux-amd64", Size: 123}}}
	if _, err := releaseForPlatform(r, "linux", "amd64"); err == nil {
		t.Fatal("accepted release without checksum")
	}
	r.Assets = append(r.Assets, githubAsset{ID: 2, Name: "komari-agent-linux-amd64.sha256"})
	got, err := releaseForPlatform(r, "linux", "amd64")
	if err != nil || got.ValidationAssetID != 2 || got.RepoOwner != "berry-shake" {
		t.Fatalf("release=%+v error=%v", got, err)
	}
	if _, err := releaseForPlatform(r, "linux", "arm64"); err == nil {
		t.Fatal("accepted wrong architecture")
	}
}
