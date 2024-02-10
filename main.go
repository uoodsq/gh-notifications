package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/adrg/xdg"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/browser"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/urfave/cli"
)

var (
	client    *api.RESTClient
	storePath = filepath.Join(xdg.DataHome, "gh-notifications/store.json")
)

func init() {
	var err error

	client, err = api.DefaultRESTClient()

	if err != nil {
		log.Fatal(err)
	}
}

type Repository struct {
	FullName string `json:"full_name"`
	HtmlUrl  string `json:"html_url"`
}

type Subject struct {
	Title string `json:"title"`
	Url   string `json:"url"`
}

type Details struct {
	HtmlUrl string `json:"html_url"`
}

func (s Subject) GetDetails() (*Details, error) {
	var d Details

	log.Printf("loading details for '%s'", s.Title)

	err := client.Get(s.Url, &d)

	if err != nil {
		return nil, err
	}

	return &d, nil
}

type Notification struct {
	Id         string     `json:"id"`
	Repository Repository `json:"repository"`
	Subject    Subject    `json:"subject"`
}

type Store struct {
	Notifications map[string]Notification
	IgnoredRepos  map[string]bool
	Details       map[string]*Details
}

func NewStore() *Store {
	return &Store{
		Notifications: make(map[string]Notification),
		IgnoredRepos:  make(map[string]bool),
		Details:       make(map[string]*Details),
	}
}

func LoadStore() (*Store, error) {
	f, err := os.Open(storePath)

	if errors.Is(err, os.ErrNotExist) {
		store := NewStore()

		return store, nil
	}

	if err != nil {
		return nil, err
	}

	var store Store

	err = json.NewDecoder(f).Decode(&store)

	if err != nil {
		return nil, err
	}

	if store.Notifications == nil {
		store.Notifications = make(map[string]Notification)
	}

	if store.IgnoredRepos == nil {
		store.IgnoredRepos = make(map[string]bool)
	}

	if store.Details == nil {
		store.Details = make(map[string]*Details)
	}

	return &store, nil
}

func (s *Store) Dump() error {
	err := os.MkdirAll(filepath.Dir(storePath), 0700)

	if err != nil {
		return err
	}

	f, err := os.Create(storePath)

	if err != nil {
		return err
	}

	err = json.NewEncoder(f).Encode(&s)

	if err != nil {
		return err
	}

	return nil
}

func (s Store) GetDetails(n Notification) (*Details, error) {
	if d, ok := s.Details[n.Id]; ok {
		return d, nil
	}

	d, err := n.Subject.GetDetails()

	if err != nil {
		return nil, err
	}

	s.Details[n.Id] = d

	return d, nil
}

func (s Store) RenderIgnoredRepos() {
	repos := make([]string, 0)

	for repo := range s.IgnoredRepos {
		repos = append(repos, repo)
	}

	sort.Strings(repos)

	t := table.NewWriter()
	t.SetStyle(table.StyleRounded)
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Ignored Repo"})

	for _, repo := range repos {
		url := fmt.Sprintf("https://github.com/%s", repo)

		t.AppendRow(table.Row{text.Hyperlink(url, repo)})
	}

	t.Render()
}

func (s Store) RenderNotifications() error {
	notificationsByRepo := s.NotificationsByRepo()

	repos := make([]Repository, 0)

	for repo := range notificationsByRepo {
		repos = append(repos, repo)
	}

	sort.SliceStable(repos, func(i, j int) bool {
		return strings.Compare(repos[i].FullName, repos[j].FullName) == -1
	})

	t := table.NewWriter()
	t.SetStyle(table.StyleRounded)
	t.SetOutputMirror(os.Stdout)
	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Number:      1,
			Align:       text.AlignRight,
			AlignHeader: text.AlignRight,
		},
	})
	t.AppendHeader(table.Row{"Repo", "ID", "Title"})

	for _, repo := range repos {
		for i, n := range notificationsByRepo[repo] {
			r := ""

			if i == 0 {
				r = text.Hyperlink(n.Repository.HtmlUrl, n.Repository.FullName)
			}

			d, err := s.GetDetails(n)

			if err != nil {
				return err
			}

			id := text.Hyperlink(d.HtmlUrl, n.Id)

			t.AppendRow(table.Row{
				r,
				id,
				n.Subject.Title,
			})
		}

		t.AppendSeparator()
	}

	t.Render()

	return nil
}

func (s Store) NotificationsForRepo(repo string) []Notification {
	m := make([]Notification, 0)

	for _, n := range s.Notifications {
		if n.Repository.FullName == repo {
			m = append(m, n)
		}
	}

	return m
}

func (s Store) NotificationsByRepo() map[Repository][]Notification {
	m := make(map[Repository][]Notification)

	for _, n := range s.Notifications {
		m[n.Repository] = append(m[n.Repository], n)
	}

	return m
}

func (s *Store) Done(n Notification) error {
	if n.Subject.Title == "" {
		log.Printf("marking notification %s done", n.Id)
	} else {
		log.Printf("marking '%s' done", n.Subject.Title)
	}

	if err := client.Delete(fmt.Sprintf("notifications/threads/%s", n.Id), &struct{}{}); err != nil {
		return err
	}

	delete(s.Notifications, n.Id)

	return nil
}

func (s Store) Open(n Notification) error {
	b := browser.New("", os.Stdout, os.Stderr)

	d, err := n.Subject.GetDetails()

	if err != nil {
		return err
	}

	err = b.Browse(d.HtmlUrl)

	return err
}

func syncAction(c *cli.Context) error {
	store, err := LoadStore()

	if err != nil {
		return err
	}

	var notifications []Notification

	if err := client.Get("notifications", &notifications); err != nil {
		return err
	}

	for _, n := range notifications {
		if store.IgnoredRepos[n.Repository.FullName] {
			store.Done(n)
		} else {
			store.Notifications[n.Id] = n
		}
	}

	if err := store.Dump(); err != nil {
		return err
	}

	return listAction(c)
}

func listAction(c *cli.Context) error {
	store, err := LoadStore()

	if err != nil {
		return err
	}

	store.RenderNotifications()

	err = store.Dump()

	if err != nil {
		return err
	}

	return nil
}

func ignoreAction(c *cli.Context) error {
	store, err := LoadStore()

	if err != nil {
		return err
	}

	for _, repo := range c.Args() {
		store.IgnoredRepos[repo] = true

		for _, n := range store.NotificationsForRepo(repo) {
			if err := store.Done(n); err != nil {
				return err
			}
		}
	}

	if err := store.Dump(); err != nil {
		return err
	}

	return ignoredAction(c)
}

func unignoreAction(c *cli.Context) error {
	store, err := LoadStore()

	if err != nil {
		return err
	}

	for _, repo := range c.Args() {
		delete(store.IgnoredRepos, repo)
	}

	if err := store.Dump(); err != nil {
		return err
	}

	return ignoredAction(c)
}

func ignoredAction(c *cli.Context) error {
	store, err := LoadStore()

	if err != nil {
		return err
	}

	store.RenderIgnoredRepos()

	return nil
}

func doneAction(c *cli.Context) error {
	store, err := LoadStore()

	if err != nil {
		return err
	}

	for _, id := range c.Args() {
		n := Notification{Id: id}

		err := store.Done(n)

		if err != nil {
			return err
		}
	}

	if err := store.Dump(); err != nil {
		return err
	}

	return listAction(c)
}

func main() {
	app := &cli.App{
		Name:  "gh-notifications",
		Usage: "wrangle your github inbox",
		Commands: []cli.Command{
			{
				Name:        "sync",
				Usage:       "sync notifications to disk",
				Description: "Pull new notifications from GitHub to disk.  Any notifications belonging to repos that are marked ignored are automatically dismissed",
				Action:      syncAction,
			},
			{
				Name:        "list",
				Usage:       "list synced notifications",
				Description: "Show all notifications currently on disk.",
				Action:      listAction,
			},
			{
				Name:        "ignore",
				Usage:       "ignore notifications from named repos",
				Description: "Mark a repo as ignored.  All notifications on disk belonging to the repo are dismissed.  Any future notifications synced from GitHub are automatically dismissed.",
				Action:      ignoreAction,
			},
			{
				Name:        "unignore",
				Usage:       "unignore notifications from named repos",
				Description: "Remove a repo from the ignored list.  Any future notifications synced from GitHub will no longer be automatically dismissed.",
				Action:      unignoreAction,
			},
			{
				Name:        "ignored",
				Usage:       "display ignored repos",
				Description: "Display the list of ignored repos.",
				Action:      ignoredAction,
			},
			{
				Name:        "done",
				Usage:       "mark notification done",
				Description: "Dismiss a notification by its ID.",
				Action:      doneAction,
			},
			{
				Name:        "reset",
				Usage:       "delete cached notifications",
				Description: "Delete the local data file, which includes the list of ignored repos and any synced notifications.  Only do this if you're having problems.",
				Action: func(c *cli.Context) error {
					return os.Remove(storePath)
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
