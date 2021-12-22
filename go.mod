module github.com/Dynom/ERI

go 1.14

require (
	cloud.google.com/go/pubsub v1.17.1
	github.com/BurntSushi/toml v0.4.1
	github.com/Dynom/TySug v0.1.4
	github.com/NYTimes/gziphandler v1.1.1
	github.com/Pimmr/rig v1.0.6
	github.com/cncf/xds/go v0.0.0-20211216145620-d92e9ce0af51 // indirect
	github.com/graphql-go/graphql v0.8.0
	github.com/graphql-go/handler v0.2.3
	github.com/juju/ratelimit v1.0.1
	github.com/lib/pq v1.10.4
	github.com/minio/highwayhash v1.0.2
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rs/cors v1.8.2
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.3.0
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	golang.org/x/net v0.0.0-20211216030914-fe4d6282115f
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
	google.golang.org/api v0.63.0
	google.golang.org/genproto v0.0.0-20211221231510-d629cc9a93d5 // indirect
	google.golang.org/grpc v1.43.0 // indirect
)

//replace github.com/Dynom/TySug => ../TySug
