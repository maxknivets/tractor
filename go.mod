module github.com/manifold/tractor

go 1.12

require (
	github.com/armon/circbuf v0.0.0-20190214190532-5111143e8da2
	github.com/d5/tengo v1.24.3
	github.com/davecgh/go-spew v1.1.1
	github.com/daviddengcn/go-colortext v0.0.0-20180409174941-186a3d44e920
	github.com/dustin/go-jsonpointer v0.0.0
	github.com/fsnotify/fsnotify v1.4.7
	github.com/getlantern/systray v0.0.0-20191210013027-82c477f5e254
	github.com/getlantern/uuid v1.2.0 // indirect
	github.com/gliderlabs/com v0.1.1-0.20191023181249-02615ad445ac
	github.com/gliderlabs/stdcom v0.0.0-20171109193247-64a0d4e5fd86
	github.com/goji/httpauth v0.0.0-20160601135302-2da839ab0f4d
	github.com/golangplus/testing v0.0.0-20180327235837-af21d9c3145e // indirect
	github.com/hashicorp/yamux v0.0.0-20190923154419-df201c70410d // indirect
	github.com/inconshreveable/muxado v0.0.0-20160802230925-fc182d90f26e // indirect
	github.com/keybase/go-ps v0.0.0-20190827175125-91aafc93ba19
	github.com/lucas-clemente/quic-go v0.13.1 // indirect
	github.com/lxn/walk v0.0.0-20191128110447-55ccb3a9f5c1 // indirect
	github.com/lxn/win v0.0.0-20191128105842-2da648fda5b4 // indirect
	github.com/manifold/qtalk v0.0.0-20191117202844-f1ce2a287d67
	github.com/mitchellh/hashstructure v1.0.0
	github.com/mitchellh/mapstructure v1.1.2
	github.com/nickvanw/ircx/v2 v2.0.0
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/progrium/prototypes v0.0.0-20190807232325-d9b2b4ba3a4f
	github.com/rjeczalik/notify v0.9.2
	github.com/rs/xid v1.2.1
	github.com/skratchdot/open-golang v0.0.0-20190402232053-79abb63cd66e
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.4.0
	github.com/thejerf/suture v3.0.2+incompatible
	github.com/urfave/negroni v1.0.0
	go.uber.org/atomic v1.5.1 // indirect
	go.uber.org/multierr v1.4.0 // indirect
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20191117063200-497ca9f6d64f
	golang.org/x/lint v0.0.0-20191125180803-fdd1cda4f05f // indirect
	golang.org/x/net v0.0.0-20191118183410-d06c31c94cae // indirect
	golang.org/x/sys v0.0.0-20191210023423-ac6580df4449 // indirect
	golang.org/x/tools v0.0.0-20191205215504-7b8c8591a921 // indirect
	google.golang.org/appengine v1.6.5 // indirect
	gopkg.in/sorcix/irc.v2 v2.0.0-20190306112350-8d7a73540b90
	gopkg.in/vmihailenco/msgpack.v2 v2.9.1 // indirect
)

replace github.com/dustin/go-jsonpointer => ./vnd/github.com/dustin/go-jsonpointer
