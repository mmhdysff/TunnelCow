package tunnel

const (
	DefaultControlPort = 64290

	DefaultDashboardPort = 10000
)

type Config struct {
	ServerAddr string
	Token      string
}
