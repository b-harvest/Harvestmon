package util

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func Test(t *testing.T) {

	t.Run("github directory", func(t *testing.T) {
		res, err := FetchGithubFile(os.Getenv("OWNER"), os.Getenv("REPO"), os.Getenv("BRANCH"), os.Getenv("PATH"), os.Getenv("GITHUB_TOKEN"))
		assert.Nil(t, err)

		for _, r := range res {
			fmt.Printf("%s\n=======\n", r)
		}

	})

}
