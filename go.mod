module github.com/Dynom/ERI

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/Dynom/TySug v0.1.3-0.20190501140824-4748e35329ec
	github.com/NYTimes/gziphandler v1.0.1
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e
	github.com/graphql-go/graphql v0.7.9
	github.com/graphql-go/handler v0.2.3
	github.com/juju/ratelimit v1.0.1
	github.com/kr/pretty v0.2.0 // indirect
	github.com/lib/pq v1.3.0
	github.com/minio/highwayhash v1.0.0
	github.com/prometheus/common v0.9.1 // indirect
	github.com/sirupsen/logrus v1.4.2
	go.undefinedlabs.com/scopeagent v0.0.0-20200123124745-640276f81881 // indirect
)

replace github.com/Dynom/TySug => ../TySug
