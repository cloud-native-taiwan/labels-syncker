package main

import (
	"context"
	goflag "flag"
	"io/ioutil"
	"net/http"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/glog"
	"github.com/google/go-github/v27/github"
	flag "github.com/spf13/pflag"
	"github.com/thoas/go-funk"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

var (
	token  string
	config string
	ctx    = context.Background()
)

type Label struct {
	Name               string   `yaml:"name"`
	Description        string   `yaml:"description"`
	Color              string   `yaml:"color"`
	Repositories       []string `yaml:"repositories"`
	IgnoreRepositories []string `yaml:"ignoreRepositories"`
}

type Config struct {
	Fork   bool    `yaml:"fork"`
	Labels []Label `yaml:"labels"`
}

type Configs map[string]Config

func parseFlags() {
	flag.StringVarP(&config, "config", "", "labels.yml", "Absolute path to labels config file.")
	flag.StringVarP(&token, "token", "", "", "Your access token for GitHub.")
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()
}

func loadConfig() (Configs, error) {
	b, err := ioutil.ReadFile(config)
	if err != nil {
		return nil, err
	}

	var cfgs Configs
	if err := yaml.Unmarshal(b, &cfgs); err != nil {
		return nil, err
	}
	return cfgs, nil
}

func listLabels(client *github.Client, org, repo string) (mapset.Set, error) {
	opt := &github.ListOptions{}
	labels, _, err := client.Issues.ListLabels(context.Background(), org, repo, opt)
	if err != nil {
		return nil, err
	}

	newLabels := mapset.NewSet()
	for _, l := range labels {
		newLabels.Add(l.GetName())
	}
	return newLabels, nil
}

func createOrUpdateLabels(client *github.Client, org, repo string, cfg Config, expectedLabels mapset.Set) error {
	for _, l := range cfg.Labels {
		if expectedLabels.Contains(l.Name) {
			label := &github.Label{
				Name:        &l.Name,
				Description: &l.Description,
				Color:       &l.Color,
			}

			glog.Infof("%s/%s: Creating or updating \"%s\" label ...", org, repo, l.Name)
			_, resp, _ := client.Issues.GetLabel(ctx, org, repo, l.Name)
			switch resp.StatusCode {
			case http.StatusOK:
				_, _, err := client.Issues.EditLabel(ctx, org, repo, l.Name, label)
				if err != nil {
					glog.Fatalf("%s/%s: Failed to update label, %+v.", org, repo, err)
				}
			case http.StatusNotFound:
				_, _, err := client.Issues.CreateLabel(ctx, org, repo, label)
				if err != nil {
					glog.Fatalf("%s/%s: Failed to create label, %+v.", org, repo, err)
				}
			}
			glog.Infof("%s/%s: The \"%s\" label has been done.", org, repo, l.Name)
		}
	}
	return nil
}

func deleteLabels(client *github.Client, org, repo string, labels mapset.Set) error {
	for _, l := range labels.ToSlice() {
		r, err := client.Issues.DeleteLabel(context.Background(), org, repo, l.(string))
		if err != nil && r.StatusCode != 204 {
			return err
		}
	}
	return nil
}

func main() {
	defer glog.Flush()
	parseFlags()

	cfgs, err := loadConfig()
	if err != nil {
		glog.Fatalln(err)
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	for org := range cfgs {
		cfg := cfgs[org]
		repos, _, err := client.Repositories.List(ctx, org, nil)
		if err != nil {
			glog.Fatalln(err)
		}

		for _, repo := range repos {
			if !cfg.Fork && *repo.Fork {
				continue
			}

			expectedLabels := mapset.NewSet()
			for _, l := range cfg.Labels {
				if funk.ContainsString(l.IgnoreRepositories, *repo.Name) {
					continue
				}

				if len(l.Repositories) > 0 {
					if funk.ContainsString(l.Repositories, *repo.Name) {
						expectedLabels.Add(l.Name)
					}
					continue
				}
				expectedLabels.Add(l.Name)
			}

			if len(expectedLabels.ToSlice()) > 0 {
				labels, err := listLabels(client, org, *repo.Name)
				if err != nil {
					glog.Fatalln(err)
				}

				cleanLabels := labels.Difference(expectedLabels)
				if err := deleteLabels(client, org, *repo.Name, cleanLabels); err != nil {
					glog.Fatalln(err)
				}

				if err := createOrUpdateLabels(client, org, *repo.Name, cfg, expectedLabels); err != nil {
					glog.Fatalln(err)
				}
			}
		}
	}
}
