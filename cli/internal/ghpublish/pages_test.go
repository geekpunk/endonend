package ghpublish

import "testing"

func TestParsePagesURL(t *testing.T) {
	tests := []struct {
		name        string
		identityURL string
		wantOwner   string
		wantRepo    string
		wantUser    bool
		wantErr     bool
	}{
		{
			name:        "user page root",
			identityURL: "https://geekpunk.github.io",
			wantOwner:   "geekpunk", wantRepo: "geekpunk.github.io", wantUser: true,
		},
		{
			name:        "user page root with trailing slash",
			identityURL: "https://geekpunk.github.io/",
			wantOwner:   "geekpunk", wantRepo: "geekpunk.github.io", wantUser: true,
		},
		{
			name:        "project page",
			identityURL: "https://someartist.github.io/bandname",
			wantOwner:   "someartist", wantRepo: "bandname", wantUser: false,
		},
		{
			name:        "project page with deeper path",
			identityURL: "https://someartist.github.io/bandname/sub/path",
			wantOwner:   "someartist", wantRepo: "bandname", wantUser: false,
		},
		{
			name:        "not a github.io URL",
			identityURL: "https://ligatures.example",
			wantErr:     true,
		},
		{
			name:        "github.io with no owner",
			identityURL: "https://.github.io/x",
			wantErr:     true,
		},
		{
			name:        "unparseable URL",
			identityURL: "://not a url",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePagesURL(tt.identityURL)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParsePagesURL(%q): want error, got %+v", tt.identityURL, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParsePagesURL(%q): unexpected error: %v", tt.identityURL, err)
			}
			if got.Owner != tt.wantOwner || got.Repo != tt.wantRepo || got.IsUserPage != tt.wantUser {
				t.Errorf("ParsePagesURL(%q) = %+v, want owner=%s repo=%s isUserPage=%v",
					tt.identityURL, got, tt.wantOwner, tt.wantRepo, tt.wantUser)
			}
		})
	}
}

func TestPagesTarget_URL(t *testing.T) {
	userPage := &PagesTarget{Owner: "geekpunk", Repo: "geekpunk.github.io", IsUserPage: true}
	if got := userPage.URL(); got != "https://geekpunk.github.io" {
		t.Errorf("user page URL = %q, want https://geekpunk.github.io", got)
	}

	projectPage := &PagesTarget{Owner: "someartist", Repo: "bandname", IsUserPage: false}
	if got := projectPage.URL(); got != "https://someartist.github.io/bandname" {
		t.Errorf("project page URL = %q, want https://someartist.github.io/bandname", got)
	}
}

func TestPagesTarget_FullName(t *testing.T) {
	target := &PagesTarget{Owner: "someartist", Repo: "bandname"}
	if got := target.FullName(); got != "someartist/bandname" {
		t.Errorf("FullName() = %q, want someartist/bandname", got)
	}
}
