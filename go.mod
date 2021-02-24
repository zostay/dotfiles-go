module github.com/zostay/dotfiles-go

go 1.14

replace github.com/zostay/go-email => ../go-email

replace github.com/zostay/go-addr => ../go-addr

require (
	github.com/ansd/lastpass-go v0.1.1
	github.com/araddon/dateparse v0.0.0-20201001162425-8aadafed4dc4
	github.com/bbrks/wrap v2.3.0+incompatible
	github.com/emersion/go-message v0.14.1
	github.com/emersion/go-sasl v0.0.0-20200509203442-7bfe0ed36a21
	github.com/emersion/go-smtp v0.14.0
	github.com/gopasspw/gopass v1.10.1
	github.com/joho/godotenv v1.3.0
	github.com/kr/pretty v0.2.1
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.7.0
	github.com/tobischo/gokeepasslib/v3 v3.1.0
	github.com/zalando/go-keyring v0.1.0
	github.com/zostay/go-addr v0.0.0-20210219053543-2c79a4ed235b
	github.com/zostay/go-email v0.0.0-20210220050615-4e0916087ed6
	github.com/zostay/go-esv-api v0.0.0-20201114154340-be89d3d9bb0c
	golang.org/x/net v0.0.0-20201021035429-f5854403a974 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
)
