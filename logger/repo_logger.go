// Packager logger ...
package logger

type repoLogger struct {
	repo, pkg string
}

func GetRepoLog(repo, pkg string) {
	return &repLogger{
		repo: repo,
		pkg:  pkg,
	}
}

func (r *repoLogger) Debug(args ...interface{}) {
	if len(args) == 1 {
		record := fmt.Sprintf("%s.%s %v", r.repo, r.pkg, args[0])
		gLog.Debug(record)
		return
	}
	gLog.Debug(args)
}
