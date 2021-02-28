package mail

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ActionsSummary is the summary of actions taken while filtering to display to
// the user.
type ActionsSummary map[string]int

// String returns a nice tabular summary of the actions to the console.
//  fmt.Print(action)
func (actions ActionsSummary) String() string {
	var out strings.Builder

	total := 0
	kw := 5
	cw := 1
	keys := make([]string, 0, len(actions))
	for key, count := range actions {
		total += count
		keys = append(keys, key)

		if len(key) > kw {
			kw = len(key)
		}

		countLen := len(strconv.Itoa(count))
		if countLen > cw {
			cw = countLen
		}
	}

	sort.Strings(keys)

	kws := strconv.Itoa(kw)
	cws := strconv.Itoa(cw)

	if total > 0 {
		for _, key := range keys {
			fmt.Fprintf(&out, " %-"+kws+"s : %"+cws+"d\n", key, actions[key])
		}

		fmt.Fprintf(&out, "%s %s\n", strings.Repeat("-", kw+2), strings.Repeat("-", cw+2))
		fmt.Fprintf(&out, " %-"+kws+"s : %"+cws+"d\n", "Total", total)
	} else {
		fmt.Fprintln(&out, "Nothing to do.")
	}

	return out.String()
}
