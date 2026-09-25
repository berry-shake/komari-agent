package update

import "testing"

func TestDistributionReleaseSelection(t *testing.T) {
	releases := []githubRelease{
		{TagName: "1.2.13-fork.2"}, {TagName: "1.2.60-fork.99"},
		{TagName: "1.2.3"}, {TagName: "1.2.4"}, {TagName: "1.2.9"}, {TagName: "1.2.10"},
		{TagName: "2.0.0", Draft: true}, {TagName: "2.1.0", Prerelease: true},
		{TagName: "1.3.0-beta.1"}, {TagName: "v1.3.0"}, {TagName: "1.3.0+build"}, {TagName: "1.03.0"},
	}
	if got := selectDistributionRelease(releases); got == nil || got.TagName != "1.2.10" {
		t.Fatalf("selected %+v", got)
	}
	if got := selectDistributionRelease(releases[:3]); got != nil {
		t.Fatalf("accepted legacy tag: %+v", got)
	}
	if !isDistributionVersion("1.2.4") || !isDistributionVersion("2.0.0") {
		t.Fatal("numeric release rejected")
	}
}
func TestPlatformReleaseRequiresMatchingChecksum(t *testing.T) {
	r := &githubRelease{TagName: "1.2.4", Assets: []githubAsset{{ID: 1, Name: "komari-agent-linux-amd64", Size: 123}}}
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
