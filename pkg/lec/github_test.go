package lec

import "testing"

func TestRepositoryBoundary(t *testing.T) {
	t.Setenv("LEC_GITHUB_ORG", "lec-org")
	for _, value := range []string{"https://github.com/lec-org/lec-oj.git", "git@github.com:lec-org/LecPunch.git", "ssh://git@github.com/LEC-ORG/devtron.git"} {
		if err := ValidateRepository(value); err != nil {
			t.Fatalf("%s: %v", value, err)
		}
	}
	for _, value := range []string{"https://github.com/other/repo", "https://github.com.evil/lec-org/a", "https://user:token@github.com/lec-org/a", "https://github.com/lec-org/../a", "https://github.com/lec-org/%61", "http://github.com/lec-org/a", "ssh://root@github.com/lec-org/a", "file:///lec-org/a", "https://github.com/lec-org/a?x=1"} {
		if err := ValidateRepository(value); err == nil {
			t.Fatalf("接受了非法仓库 %s", value)
		}
	}
}
