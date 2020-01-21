module github.com/manifold/tractor/dev/workspace

go 1.13

replace workspace => ./

require github.com/manifold/tractor v0.0.0

replace github.com/manifold/tractor => ../..

replace github.com/dustin/go-jsonpointer => ../../vnd/github.com/dustin/go-jsonpointer
