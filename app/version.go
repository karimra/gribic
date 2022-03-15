package app

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
)

var (
	version     = "dev"
	commit      = "none"
	date        = "unknown"
	gitURL      = ""
	downloadURL = "https://github.com/karimra/gribic/raw/main/install.sh"
)

func (a *App) RunEVersion(cmd *cobra.Command, args []string) error {
	fmt.Printf("version : %s\n", version)
	fmt.Printf(" commit : %s\n", commit)
	fmt.Printf("   date : %s\n", date)
	fmt.Printf(" gitURL : %s\n", gitURL)
	fmt.Printf("   docs : https://gribic.kmrd.dev\n")
	return nil
}

func (a *App) VersionUpgradeRun(cmd *cobra.Command, args []string) error {
	f, err := ioutil.TempFile("", "gribic")
	defer os.Remove(f.Name())
	if err != nil {
		return err
	}
	err = downloadFile(downloadURL, f)
	if err != nil {
		return err
	}

	var c *exec.Cmd
	switch a.Config.LocalFlags.UpgradeUsePkg {
	case true:
		c = exec.Command("bash", f.Name(), "--use-pkg")
	case false:
		c = exec.Command("bash", f.Name())
	}

	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err = c.Run()
	if err != nil {
		return err
	}
	return nil
}

// downloadFile will download a file from a URL and write its content to a file
func downloadFile(url string, file *os.File) error {
	client := http.Client{Timeout: 10 * time.Second}
	// Get the data
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}
	return nil
}
