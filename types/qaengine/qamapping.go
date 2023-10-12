package qaengine

import (
	"regexp"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/sirupsen/logrus"
)

// QAMapping defines the structure of the QA Category mapping file
type QAMapping struct {
	Categories []QACategory `yaml:"categories"`
}

// QACategory represents a category of QA engine questions
type QACategory struct {
	Name      string   `yaml:"name"`
	Enabled   bool     `yaml:"enabled"`
	Questions []string `yaml:"questions"`
}

// GetProblemCategories returns the list of categories a particular question belongs to
func GetProblemCategories(targetProbId string) []string {
	var categories []string
	for category, probIds := range common.QACategoryMap {
		for _, probId := range probIds {
			// if the problem ID contains a capture group (\), it's a regular expression
			if strings.ContainsAny(probId, "(\\)") {
				re, err := regexp.Compile(probId)
				if err != nil {
					logrus.Errorf("invalid problem ID regexp: %s\n", targetProbId)
				}
				if re.Match([]byte(targetProbId)) {
					categories = append(categories, category)
				}
			} else if targetProbId == probId {
				categories = append(categories, category)
			}
		}

	}

	if len(categories) == 0 {
		categories = append(categories, "default")
	}

	return categories
}
