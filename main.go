package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rompei/inco"
	"github.com/darfk/ts3"
	"github.com/hashicorp/go-version"
)

// VersionURL is url for source of ts3 versions.
const VersionURL = "https://www.server-residenz.com/tools/ts3versions.json"

// RMCommand is command to remove docker machine.
const RMCommand = "docker rm -f %s"

// RunCommand is command to run docker machine.
const RunCommand = "docker run -d -p 9987:9987/udp -p 10011:10011 -p 30033:30033 -v %s:/data --name=%s aheil/teamspeak3-server"

// TS3DBName is name of database for TS3
const TS3DBName = "ts3server.sqlitedb"

// VersionInfo is object of response of ts3 servers.
type VersionInfo struct {
	Checked  string   `json:"checked"`
	Latest   string   `json:"latest"`
	Versions []string `json:"versions"`
}

func main() {

	var (
		containerName   string
		dataDir         string
		backupDir       string
		notificationURL string
	)

	flag.StringVar(&containerName, "c", "ts3-server", "Container name.")
	flag.StringVar(&dataDir, "d", "", "Data directory on host.")
	flag.StringVar(&backupDir, "b", "", "Backup directory.")
	flag.StringVar(&notificationURL, "n", "", "Slack notification url.")
	flag.Parse()

	// File paths must be absolute path.
	if (dataDir != "" && !filepath.IsAbs(dataDir)) || (backupDir != "" && !filepath.IsAbs(backupDir)) {
		log.Fatal("dataDir and backupDir is must be absolute path.")
	}

	// Connect to TeamSpeak3 Server Query.
	client, err := ts3.NewClient(":10011")
	if err != nil {
		log.Println(err)
	}
	defer client.Close()

	// Getting current version.
	cv := getCurrentVersion(client)

	// Getting ts3 versions.
	versions, err := getTS3Versions()
	if err != nil {
		log.Fatal(err)
	}

	// Debug
	//cv = "3.0.13.5"
	currentVersion, err := version.NewVersion(cv)
	if err != nil {
		log.Fatal(err)
	}
	latestVersion, err := version.NewVersion(versions.Latest)
	if err != nil {
		log.Fatal(err)
	}

	if currentVersion.LessThan(latestVersion) {
		// outdated
		if backupDir != "" {
			// If backupDir is set, backup database.
			if err = backupDB(dataDir, backupDir); err != nil {
				log.Fatal(err)
			}
		}

		// Remove container and recreate.
		log.Printf("Update to version %s", latestVersion)
		out, err := rmDocker(containerName)
		if err != nil {
			log.Printf(out)
			log.Fatal(err)
		}
		log.Print(out)
		out, err = runDocker(containerName, dataDir)
		if err != nil {
			log.Printf(out)
			log.Fatal(err)
		}
		log.Printf(out)

		if notificationURL != "" {
			// If notificationURL is set, notify to Slack.
			notifySlack(notificationURL, "TeamSpeak3 server was updated automatically.")
		}
	}
}

func backupDB(dataDir, backupDir string) error {
	dbPath := filepath.Join(dataDir, TS3DBName)
	if _, err := os.Stat(backupDir); err != nil {
		if err = os.MkdirAll(backupDir, 0755); err != nil {
			return err
		}
	}
	src, err := os.Open(dbPath)
	if err != nil {
		return err
	}
	defer src.Close()

	backupName := "ts3db_" + time.Now().Format("2006_01_02_15_04_05") + ".sqlitedb"
	dst, err := os.Create(filepath.Join(backupDir, backupName))
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func notifySlack(notificationURL, msg string) error {
	e := &inco.Message{
		Text:     msg,
		Channel:  "#test",
		Username: "TS3Updater",
	}
	if err := inco.Incoming(notificationURL, e); err != nil {
		return err
	}
	return nil
}

func rmDocker(name string) (string, error) {
	command := strings.Fields(fmt.Sprintf(RMCommand, name))
	log.Println(command)
	out, err := exec.Command(command[0], command[1:]...).Output()
	return string(out), err
}

func runDocker(name string, dataDir string) (string, error) {
	command := strings.Fields(fmt.Sprintf(RunCommand, dataDir, name))
	log.Println(command)
	out, err := exec.Command(command[0], command[1:]...).Output()
	return string(out), err
}

func getCurrentVersion(client *ts3.Client) string {
	r, err := client.Exec(ts3.Version())
	if err != nil {
		log.Fatal(err)
	}
	return r.Params[0]["version"]
}

func getTS3Versions() (*VersionInfo, error) {
	resp, err := http.Get(VersionURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var info VersionInfo
	if err = json.Unmarshal(b, &info); err != nil {
		return nil, err
	}
	return &info, nil
}
