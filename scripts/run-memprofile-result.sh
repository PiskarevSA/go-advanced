go test github.com/PiskarevSA/go-advanced/internal/usecases -bench=. -memprofile=profiles/result.pprof
mv usecases.test usecases-result.test