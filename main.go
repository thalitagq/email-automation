package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	c, err := client.DialTLS(os.Getenv("IMAP_SERVER"), nil)
	if err != nil {
		fmt.Println(err)
	}
	defer c.Logout()

	if err := c.Login(os.Getenv("EMAIL"), os.Getenv("PASSWORD")); err != nil {
		fmt.Println(err)
	}

	mbox, err := c.Select("INBOX.Curriculos", false)
	if err != nil {
		fmt.Println(err)
	}

	if mbox.Messages > 0 {
		from := uint32(1)
		to := mbox.Messages

		seqSet := new(imap.SeqSet)
		seqSet.AddRange(from, to)

		messages := make(chan *imap.Message)
		section := &imap.BodySectionName{}
		items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchRFC822, section.FetchItem()}

		go func() {
			if err := c.Fetch(seqSet, items, messages); err != nil {
				fmt.Println(err)
			}
		}()

		if err != nil {
			fmt.Printf("Failed to fetch first message in INBOX.Curriculos: %v", err)
		}

		for msg := range messages {
			if msg == nil {
				continue
			}

			r := msg.GetBody(section)
			if r == nil {
				fmt.Println("No body on messsage")
				continue
			}

			mr, err := mail.CreateReader(r)
			if err != nil {
				fmt.Println("Error creating message reader:", err)
				continue
			}

			for {
				p, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					fmt.Println("Error reading part:", err)
					continue
				}

				switch h := p.Header.(type) {
				case *mail.AttachmentHeader:
					filename, _ := h.Filename()
					fmt.Println("Downloading attachment:", filename)

					downloadFolder := "Files"

					_, err := os.Stat(downloadFolder)
					if os.IsNotExist(err) {
						err := os.MkdirAll(downloadFolder, 0755) 
						if err != nil {
							fmt.Println(err)
							return 
						}
					}

					filePath := filepath.Join(downloadFolder, sanitize(filename))

					out, err := os.Create(filePath)
					if err != nil {
						fmt.Println("Error creating file:", err)
						continue
					}

					defer out.Close()

					if _, err := io.Copy(out, p.Body); err != nil {
						fmt.Println("Error saving attachment:", err)
					}
				}
			}
		}
	}
}

func sanitize(filename string) string {
	return strings.ReplaceAll(filename, "/", "_")
}
