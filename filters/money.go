package filters

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/reidransom/jigyll/config"
)

// moneyFormatter renders prices using Shopify-style money filters.
//
// As in Shopify's Liquid, a price is an integer number of cents: a
// product.price of 1000 renders as "$10.00". The currency symbol and ISO code
// default to "$"/"USD" and may be overridden in _config.yml via the "currency"
// and "currency_symbol" keys.
type moneyFormatter struct {
	symbol       string // currency symbol, e.g. "$"
	code         string // ISO currency code, e.g. "USD"
	thousandsSep string
	decimalSep   string
}

func newMoneyFormatter(c *config.Config) *moneyFormatter {
	m := &moneyFormatter{
		symbol:       "$",
		code:         "USD",
		thousandsSep: ",",
		decimalSep:   ".",
	}
	if c != nil {
		if s, ok := c.String("currency"); ok && s != "" {
			m.code = s
		}
		if s, ok := c.String("currency_symbol"); ok && s != "" {
			m.symbol = s
		}
	}
	return m
}

// addMoneyFilters registers the Shopify-style money filters on the engine.
func (m *moneyFormatter) register(register func(string, interface{})) {
	register("money", m.money)
	register("money_with_currency", m.moneyWithCurrency)
	register("money_without_currency", m.moneyWithoutCurrency)
	register("money_without_trailing_zeros", m.moneyWithoutTrailingZeros)
}

// money formats a price with the currency symbol and two decimals: "$10.00".
func (m *moneyFormatter) money(value interface{}) string {
	return m.render(value, m.symbol, "", false)
}

// moneyWithCurrency appends the ISO currency code: "$10.00 USD".
func (m *moneyFormatter) moneyWithCurrency(value interface{}) string {
	return m.render(value, m.symbol, " "+m.code, false)
}

// moneyWithoutCurrency drops the symbol and code: "10.00".
func (m *moneyFormatter) moneyWithoutCurrency(value interface{}) string {
	return m.render(value, "", "", false)
}

// moneyWithoutTrailingZeros drops the decimals when the price is a whole
// amount ("$10"), but keeps them otherwise ("$10.99").
func (m *moneyFormatter) moneyWithoutTrailingZeros(value interface{}) string {
	return m.render(value, m.symbol, "", true)
}

// render formats a cent value as sign + symbol + amount + suffix, e.g.
// "-$5.50 USD". A non-numeric value renders as empty. When trimZeros is set,
// the decimals are dropped for whole amounts.
func (m *moneyFormatter) render(value interface{}, symbol, suffix string, trimZeros bool) string {
	cents, ok := toCents(value)
	if !ok {
		return ""
	}
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	decimals := 2
	if trimZeros && cents%100 == 0 {
		decimals = 0
	}
	return sign + symbol + m.formatCents(cents, decimals) + suffix
}

// formatCents formats a non-negative cent value with grouped thousands and the
// given number of fractional digits (0 or 2).
func (m *moneyFormatter) formatCents(cents int64, decimals int) string {
	var intPart, fracPart int64
	if decimals == 0 {
		intPart = (cents + 50) / 100 // round to nearest whole unit
	} else {
		intPart = cents / 100
		fracPart = cents % 100
	}
	s := groupThousands(strconv.FormatInt(intPart, 10), m.thousandsSep)
	if decimals > 0 {
		s += m.decimalSep + fmt.Sprintf("%02d", fracPart)
	}
	return s
}

// toCents interprets a Liquid value as an integer number of cents. It accepts
// integers, floats, and numeric strings, returning false for anything that
// isn't a parseable number (e.g. nil), so the filter renders empty.
func toCents(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		return int64(v), true
	case float32:
		return int64(math.Round(float64(v))), true
	case float64:
		return int64(math.Round(v)), true
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, false
		}
		return int64(math.Round(f)), true
	default:
		return 0, false
	}
}

// groupThousands inserts sep between groups of three digits, from the right.
func groupThousands(s, sep string) string {
	n := len(s)
	if n <= 3 {
		return s
	}
	var b strings.Builder
	pre := n % 3
	if pre > 0 {
		b.WriteString(s[:pre])
	}
	for i := pre; i < n; i += 3 {
		if b.Len() > 0 {
			b.WriteString(sep)
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}
