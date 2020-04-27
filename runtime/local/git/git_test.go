package git

import (
	"testing"
)

type parseCase struct {
	url      string
	expected *Source
}

func TestParseSource(t *testing.T) {
	cases := []parseCase{
		{
			url: "helloworld",
			expected: &Source{
				Repo:   "github.com/micro/services",
				Folder: "helloworld",
				Ref:    "latest",
			},
		},
		{
			url: "github.com/micro/services/helloworld",
			expected: &Source{
				Repo:   "github.com/micro/services",
				Folder: "helloworld",
				Ref:    "latest",
			},
		},
		{
			url: "github.com/micro/services/helloworld@v1.12.1",
			expected: &Source{
				Repo:   "github.com/micro/services",
				Folder: "helloworld",
				Ref:    "v1.12.1",
			},
		},
		{
			url: "github.com/micro/services/helloworld@branchname",
			expected: &Source{
				Repo:   "github.com/micro/services",
				Folder: "helloworld",
				Ref:    "branchname",
			},
		},
		{
			url: "github.com/crufter/reponame/helloworld@branchname",
			expected: &Source{
				Repo:   "github.com/crufter/reponame",
				Folder: "helloworld",
				Ref:    "branchname",
			},
		},
	}
	for i, c := range cases {
		result, err := ParseSource(c.url)
		if err != nil {
			t.Fatalf("Failed case %v: %v", i, err)
		}
		if result.Folder != c.expected.Folder {
			t.Fatalf("Folder does not match for '%v', expected '%v', got '%v'", i, c.expected.Folder, result.Folder)
		}
		if result.Repo != c.expected.Repo {
			t.Fatalf("Repo address does not match for '%v', expected '%v', got '%v'", i, c.expected.Repo, result.Repo)
		}
		if result.Ref != c.expected.Ref {
			t.Fatalf("Ref does not match for '%v', expected '%v', got '%v'", i, c.expected.Ref, result.Ref)
		}
	}
}

type nameCase struct {
	fileContent string
	expected    string
}

func TestServiceNameExtract(t *testing.T) {
	cases := []nameCase{
		{
			fileContent: `func main() {
			// New Service
			service := micro.NewService(
				micro.Name("go.micro.service.helloworld"),
				micro.Version("latest"),
			)`,
			expected: "go.micro.service.helloworld",
		},
	}
	for i, c := range cases {
		result := extractServiceName([]byte(c.fileContent))
		if result != c.expected {
			t.Fatalf("Case %v, expected: %v, got: %v", i, c.expected, result)
		}
	}
}
