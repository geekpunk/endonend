package ghpublish

import (
	"fmt"
	"testing"
)

func testTarget() *PagesTarget {
	return &PagesTarget{Owner: "someartist", Repo: "bandname"}
}

func TestEnsureRepo_SkipsCreateWhenRepoAlreadyExists(t *testing.T) {
	r := newFakeRunner() // "gh repo view" succeeds by default

	created, err := EnsureRepo(r, testTarget(), false)
	if err != nil {
		t.Fatalf("EnsureRepo: %v", err)
	}
	if created {
		t.Error("created = true for an already-existing repo, want false")
	}
	if r.calledWith("gh", "repo", "create", "someartist/bandname", "--public") {
		t.Error("EnsureRepo called \"gh repo create\" even though the repo already exists")
	}
}

func TestEnsureRepo_CreatesPublicRepoWhenMissing(t *testing.T) {
	r := newFakeRunner()
	r.failOn(errFake, "gh", "repo", "view", "someartist/bandname")

	created, err := EnsureRepo(r, testTarget(), false)
	if err != nil {
		t.Fatalf("EnsureRepo: %v", err)
	}
	if !created {
		t.Error("created = false, want true")
	}
	if !r.calledWith("gh", "repo", "create", "someartist/bandname", "--public") {
		t.Errorf("calls = %+v, want a \"gh repo create ... --public\"", r.calls)
	}
}

func TestEnsureRepo_CreatesPrivateRepoWhenRequested(t *testing.T) {
	r := newFakeRunner()
	r.failOn(errFake, "gh", "repo", "view", "someartist/bandname")

	if _, err := EnsureRepo(r, testTarget(), true); err != nil {
		t.Fatalf("EnsureRepo: %v", err)
	}
	if !r.calledWith("gh", "repo", "create", "someartist/bandname", "--private") {
		t.Errorf("calls = %+v, want a \"gh repo create ... --private\"", r.calls)
	}
}

func TestEnsureRepo_ReturnsErrorWhenCreateFails(t *testing.T) {
	r := newFakeRunner()
	r.failOn(errFake, "gh", "repo", "view", "someartist/bandname")
	r.failOn(errFake, "gh", "repo", "create", "someartist/bandname", "--public")

	if _, err := EnsureRepo(r, testTarget(), false); err == nil {
		t.Fatal("EnsureRepo with a failing create: want error, got nil")
	}
}

func TestPushDir_InitializesGitOnlyWhenNotAlreadyARepo(t *testing.T) {
	r := newFakeRunner() // "git rev-parse" succeeds: already a repo

	if err := PushDir(r, "/site", testTarget(), "main"); err != nil {
		t.Fatalf("PushDir: %v", err)
	}
	if r.calledWith("git", "init", "-b", "main") {
		t.Error("PushDir ran \"git init\" even though the directory was already a git repo")
	}
}

func TestPushDir_InitializesGitOnAFreshDirectory(t *testing.T) {
	r := newFakeRunner()
	r.failOn(errFake, "git", "rev-parse", "--is-inside-work-tree")

	if err := PushDir(r, "/site", testTarget(), "main"); err != nil {
		t.Fatalf("PushDir: %v", err)
	}
	if !r.calledWith("git", "init", "-b", "main") {
		t.Errorf("calls = %+v, want a \"git init -b main\"", r.calls)
	}
}

func TestPushDir_SetsRemoteAndPushes(t *testing.T) {
	r := newFakeRunner()

	if err := PushDir(r, "/site", testTarget(), "main"); err != nil {
		t.Fatalf("PushDir: %v", err)
	}
	if !r.calledWith("git", "remote", "add", "origin", "https://github.com/someartist/bandname.git") {
		t.Errorf("calls = %+v, want a \"git remote add origin ...\"", r.calls)
	}
	if !r.calledWith("git", "push", "-u", "origin", "main") {
		t.Errorf("calls = %+v, want a \"git push -u origin main\"", r.calls)
	}
}

func TestPushDir_ToleratesNoExistingRemoteToRemove(t *testing.T) {
	r := newFakeRunner()
	r.failOn(errFake, "git", "remote", "remove", "origin") // "no such remote" on a first-ever publish

	if err := PushDir(r, "/site", testTarget(), "main"); err != nil {
		t.Fatalf("PushDir: %v", err)
	}
}

func TestPushDir_SkipsCommitWhenNothingChanged(t *testing.T) {
	r := newFakeRunner() // "git status --porcelain" returns "" by default

	if err := PushDir(r, "/site", testTarget(), "main"); err != nil {
		t.Fatalf("PushDir: %v", err)
	}
	if r.calledWith("git", "commit", "-m", "Publish catalog update") {
		t.Error("PushDir committed even though status reported no changes")
	}
	if !r.calledWith("git", "push", "-u", "origin", "main") {
		t.Error("PushDir should still push even with nothing new to commit")
	}
}

func TestPushDir_ReturnsErrorWhenPushFails(t *testing.T) {
	r := newFakeRunner()
	r.failOn(errFake, "git", "push", "-u", "origin", "main")

	if err := PushDir(r, "/site", testTarget(), "main"); err == nil {
		t.Fatal("PushDir with a failing push: want error, got nil")
	}
}

func TestEnablePages_Succeeds(t *testing.T) {
	r := newFakeRunner()
	if err := EnablePages(r, testTarget(), "main"); err != nil {
		t.Fatalf("EnablePages: %v", err)
	}
	if !r.calledWith("gh", "api", "repos/someartist/bandname/pages", "-X", "POST", "-f", "build_type=legacy", "-f", "source[branch]=main", "-f", "source[path]=/") {
		t.Errorf("calls = %+v, want the pages-enable API call", r.calls)
	}
}

func TestEnablePages_ToleratesAlreadyEnabled(t *testing.T) {
	r := newFakeRunner()
	r.failOn(fmt.Errorf("gh: HTTP 409: a pages site already exists for this repository"),
		"gh", "api", "repos/someartist/bandname/pages", "-X", "POST", "-f", "build_type=legacy", "-f", "source[branch]=main", "-f", "source[path]=/")

	if err := EnablePages(r, testTarget(), "main"); err != nil {
		t.Errorf("EnablePages when Pages is already enabled: want nil, got %v", err)
	}
}

func TestEnablePages_ReturnsOtherErrors(t *testing.T) {
	r := newFakeRunner()
	r.failOn(errFake, "gh", "api", "repos/someartist/bandname/pages", "-X", "POST", "-f", "build_type=legacy", "-f", "source[branch]=main", "-f", "source[path]=/")

	if err := EnablePages(r, testTarget(), "main"); err == nil {
		t.Fatal("EnablePages with an unrelated failure: want error, got nil")
	}
}
