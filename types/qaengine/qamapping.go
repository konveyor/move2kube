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
func GetProblemCategories(probId string) []string {
	var categories []string
	for category, probIds := range common.QACategoryMap {
		for _, probId_ := range probIds {
			// if the problem ID contains a capture group (\), it's a regular expression
			if strings.ContainsAny(probId_, "(\\)") {
				re, err := regexp.Compile(probId_)
				if err != nil {
					logrus.Errorf("invalid problem ID regexp: %s\n", probId)
				}
				if re.Match([]byte(probId)) {
					categories = append(categories, category)
				}
			} else if probId == probId_ {
				categories = append(categories, category)
			}
		}

	}

	if len(categories) == 0 {
		categories = append(categories, "default")
	}

	return categories
}
