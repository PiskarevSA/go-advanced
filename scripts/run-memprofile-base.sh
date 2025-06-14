go test github.com/PiskarevSA/go-advanced/internal/usecases -bench=. -memprofile=profiles/base.pprof
mv usecases.test usecases-base.test