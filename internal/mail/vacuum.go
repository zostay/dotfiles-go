package mail

import (
	"fmt"
	"os"
	"path"
	"strings"
)

var (
	// UnwantedFolderSuffix mentions endings we want to strip during vacuuming.
	UnwantedFolderSuffix = []string{","}

	// UnwantedFolderPrefix mentions starts we want to strip during vacuuming.
	UnwantedFolderPrefix = []string{"+", "\\"}

	// UnwantedFolder mentions folders we want to strip during vacuuming.
	UnwantedFolder = []string{"[", "]", "Drafts", "Home_School", "Network", "Pseudo-Junk.Social", "Pseudo-Junk.Social_Network", "Social Network", "OtherJunk"}

	// UnwantedKeyword mentions remaps of folders we want to apply during
	// vacuuming.
	UnwantedKeyword = map[string][]string{
		"JunkSocial":                {"Network", "Pseudo-Junk.Social", "Pseudo-Junk/Social", "Psuedo-Junk/Social_Network", "Pseudo-Junk.Social_Network"},
		"Teamwork":                  {"Discussion"},
		"JunkOther":                 {"OtherJunk"},
		"Pseudo-Junk.ToDo":          {"Do", "Pseudo-Junk.To", "Pseudo-Junk.To_Do"},
		"Pseudo-Junk.Politics.Rush": {"Limbaugh", "Pseudo-Junk.Politics.Rush_Limbaugh"},
		"Jobs.GSG":                  {"Street", "Jobs.Grant_Street", "Jobs.Grant"},
		"Old.NewHope":               {"Hope", "Old.New", "Old.New_Hope"},
		"Old.JiftyBook":             {"Book", "Old.Jifty", "Old.Jifty_Book"},
		"Old.GBCWeb":                {"Web", "Old.GBC", "Old.GBC_Web"},
		"Tech.Perl.Mongers":         {"Mongers", "Tech.Perl.Perl", "Test.Perl.Perl_Mongers"},
		"Money.ForSale":             {"Sale", "Money.For", "Money.For_Sale"},
		"AccountInfo":               {"Account", "Info", "Account_Info"},
	}
)

// isUnwanted returns true if the folder matches an undesirable characteristic
// during vacuuming.
func isUnwanted(folder string) bool {
	for _, us := range UnwantedFolderSuffix {
		if strings.HasSuffix(folder, us) {
			return true
		}
	}

	for _, up := range UnwantedFolderPrefix {
		if strings.HasPrefix(folder, up) {
			return true
		}
	}

	for _, uf := range UnwantedFolder {
		if folder == uf {
			return true
		}
	}

	return false
}

// hasUnwantedKeyword returns true if the message contains an undesireable
// keyword during vacuuming.
func hasUnwantedKeyword(msg *Message) ([]string, string, error) {
	for tok, uks := range UnwantedKeyword {
		for _, uk := range uks {
			unwanted, err := msg.HasKeyword(uk)
			if unwanted || err != nil {
				return uks, tok, err
			}
		}
	}

	return []string{}, "", nil
}

// Vacuum performs the vacuum operation which attempts to clean up undesireable
// folder and keywords from my mail root.
func (fi *Filter) Vacuum() error {
	folders, err := fi.AllFolders()
	if err != nil {
		return err
	}

	for _, folder := range folders {
		if isUnwanted(folder) {
			cp.Fcolor(os.Stderr,
				"dropping", "🗑 DROPPING",
				"meh", ": ",
				"label", folder,
				"meh", "\n",
			)

			msgs, err := fi.Messages(folder)
			if err != nil {
				return err
			}

			for _, msg := range msgs {
				other, err := msg.BestAlternateFolder()
				if err != nil {
					return err
				}

				err = msg.MoveTo(fi.MailRoot, other)
				if err != nil {
					return err
				}

				err = msg.RemoveKeyword(other)
				if err != nil {
					return err
				}

				err = msg.Save()
				if err != nil {
					return err
				}

				cp.Fcolor(os.Stderr,
					"moving", "⇒ Moving ",
					"label", folder,
					"meh", " to ",
					"label", other,
					"meh", "\n",
				)
			}

			deadFolder := path.Join(fi.MailRoot, folder)
			for _, sd := range []string{"new", "cur", "tmp"} {
				err = os.Remove(path.Join(deadFolder, sd))
				if err != nil {
					cp.Fcolor(os.Stderr,
						"warn", "❗WARNING ",
						"meh", ": cannot delete ",
						"file", fmt.Sprintf("%s/%s", deadFolder, sd),
						"meh", fmt.Sprintf(": %+v\n", err),
					)
				}
			}
			err = os.Remove(deadFolder)
			if err != nil {
				cp.Fcolor(os.Stderr,
					"warn", "❗WARNING ",
					"meh", ": cannot delete ",
					"file", deadFolder,
					"meh", fmt.Sprintf(": %+v\n", err),
				)
			}
		} else {
			cp.Fcolor(os.Stderr,
				"searching", "🔍 Searching ",
				"label", folder,
				"meh", " for broken Keywords.\n",
			)

			msgs, err := fi.Messages(folder)
			if err != nil {
				return err
			}

			for _, msg := range msgs {
				change := 0

				// Cleanup unwanted chars in keywords
				nonconforming, err := msg.HasNonconformingKeywords()
				if err != nil {
					return err
				}
				if nonconforming {
					cp.Fcolor(os.Stderr,
						"fixing", "🔧 Fixing ",
						"meh", "non-conforming keywords.\n",
					)
					err := msg.CleanupKeywords()
					if err != nil {
						return err
					}
					change++
				}

				// Something went wrong somewhere
				unwanted, wanted, err := hasUnwantedKeyword(msg)
				if err != nil {
					return err
				}
				if len(unwanted) > 0 {
					cp.Fcolor(os.Stderr,
						"fixing", "🔧 Fixing ",
						"meh", "(",
						"label", strings.Join(unwanted, ", "),
						"meh", " to ",
						"label", wanted,
						"meh", ".\n",
					)
					change++
					err := msg.RemoveKeyword(unwanted...)
					if err != nil {
						return err
					}

					err = msg.AddKeyword(wanted)
					if err != nil {
						return err
					}
				}

				if change > 0 {
					err := msg.Save()
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}
