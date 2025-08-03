package period

import (
	"regexp"

	"k8s.io/utils/ptr"
)

func ParseTimeRef(timeRef *string) (string, string, string) {
	ref := ptr.Deref(timeRef, "")
	refReg := regexp.MustCompile(`^([\w\-]+)\/([\w\-]+)@([\w.\-_]+)$`)

	matches := refReg.FindStringSubmatch(ref)
	if len(matches) != 4 {
		return "", "", ""
	}

	return matches[1], matches[2], matches[3]
}
