package cmd

import (
	"fmt"
	"log"
	"github.com/spf13/cobra"
	"github.com/vrypan/fargo/fctools"
	"github.com/vrypan/fargo/config"
	"os"
	"os/user"
	"path/filepath"
	"github.com/hashicorp/go-getter"
)

var downloadCmd = &cobra.Command{
	Use:   "download [URI]",
	Aliases: []string{"d"},
	Short: "Download Farcaster-embedded URLs",
	Long: `This command works like "get", but instead
of displaying casts, it downloads the URLs embedded in
these casts.

Use the --mime-type flag to indicate the type of embedded
URLs you want to download.`,
	Run: downloadRun,
}

func downloadRun(cmd *cobra.Command, args []string) {
	fid, parts := parse_url(args)
	expandFlag, _ := cmd.Flags().GetBool("expand")
	countFlag, _ := cmd.Flags().GetUint32("count")
	grepFlag, _ := cmd.Flags().GetString("grep")
	mimetypeFlag, _ := cmd.Flags().GetString("mime-type")
	dryrunFlag, _ := cmd.Flags().GetBool("dry-run")
	skipdownloadedFlag, _ := cmd.Flags().GetBool("skip-downloaded")

	config.Load()
	download_dir := config.GetString("downloads.dir")
	if download_dir == "" {
		download_dir = "."
	}
	download_dir = normalizeLocalPath(download_dir)

	hub := fctools.NewFarcasterHub()
	defer hub.Close()

	if len(parts) == 1 && parts[0] == "casts" {
		urls := fctools.GetFidUrls(fid, countFlag, grepFlag)
		for _, u := range urls {
			m := u.UpdateContentType()
			if len(mimetypeFlag) >0 {
				if len(m) >= len(mimetypeFlag) {
					if m[0:len(mimetypeFlag)] == mimetypeFlag {
						if dryrunFlag != true {
							GetFile( u.Link, download_dir, u.Filename(), skipdownloadedFlag )
						}
						fmt.Printf("%s --> %s\n", u.Link, u.Filename() )
					}
				}
			} else {
				if dryrunFlag != true {
					GetFile( u.Link, download_dir, u.Filename(), skipdownloadedFlag )
				}
				fmt.Printf("%s --> %s\n", u.Link, u.Filename() )
			}
		}
		return
	}
	if len(parts) == 1 && parts[0][0:2] == "0x" {
		urls := fctools.GetCastUrls(fid, parts[0], expandFlag, grepFlag)
		for _, u := range urls {
			m := u.UpdateContentType()
			if len(mimetypeFlag) >0 {
				if len(m) >= len(mimetypeFlag) {
					if m[0:len(mimetypeFlag)] == mimetypeFlag {
						if dryrunFlag != true {
							GetFile( u.Link, download_dir, u.Filename(), skipdownloadedFlag )
						}
						fmt.Printf("%s --> %s\n", u.Link, u.Filename() )
					}
				}
			} else {
				if dryrunFlag != true {
					GetFile( u.Link, download_dir, u.Filename(), skipdownloadedFlag )
				}
				fmt.Printf("%s --> %s\n", u.Link, u.Filename() )
			}
		}
		return
	}
	log.Fatal("Not found")
}

func normalizeLocalPath(p string) string {
    if p[0:1] == "~" {
        usr, err := user.Current()
        if err != nil {
            log.Fatalf("%v\n",err)
        }
        home := usr.HomeDir
        return filepath.Join(home, p[1:])
    }
    return p
}

func fileExists(f string) bool {
	_, err := os.Stat(f)
	if err == nil {
		return true
	}
	return false
}

func GetFile(url string, dst_dir string, dst_file string, skipdownloadedFlag bool) string {
    if err := os.MkdirAll(dst_dir, os.ModePerm); err != nil {
        log.Fatalf("Error creating directory: %v\n", err)
    }

    path := filepath.Join(dst_dir, dst_file)
    if fileExists(path) && skipdownloadedFlag == true {
    	return path
    }
    if err := getter.GetFile(path,url); err != nil {
        log.Printf("\n%v: Error downloading file: %v\n", url, err)
        return ""
    }
    return path
}

func init() {
	rootCmd.AddCommand(downloadCmd)
	downloadCmd.Flags().BoolP("expand", "e", false, "Expand threads")
	downloadCmd.Flags().Uint32P("count", "c", 20, "Number of casts to show when getting @user/casts")
	downloadCmd.Flags().StringP("grep", "", "", "Only show casts containing a specific string")
	downloadCmd.Flags().StringP("mime-type", "", "", "Download embeds of mime/type")
	downloadCmd.Flags().BoolP("dry-run", "", false, "Do not download the files, just print the URLs and local destination")
	downloadCmd.Flags().BoolP("skip-downloaded", "", true, "If local file exists, do not download")
}