package validate

type Options struct {
	AllowShow    bool
	AllowExplain bool
}

var (
	OptionsStandard = Options{AllowExplain: true}
	OptionsMySQL    = Options{AllowShow: true, AllowExplain: true}
)
