package time_parse

import "testing"

func TestRFC3339ToCSTLayout(t *testing.T) {
	t.Log(RFC3339ToCSTLayout("2020-11-08T08:18:46+08:00"))
}

func TestCSTLayoutString(t *testing.T) {
	t.Log(CSTLayoutString())
}

func TestCSTLayoutStringToUnix(t *testing.T) {
	t.Log(CSTLayoutStringToUnix("2020-01-24 21:11:11"))
}

func TestGMTLayoutString(t *testing.T) {
	t.Log(GMTLayoutString())
}

func TestGetUnixS(t *testing.T) {
	t.Log(GetUnixS())
}

func TestGetUnixMS(t *testing.T) {
	t.Log(GetUnixMS())
}

func TestGetUnixUS(t *testing.T) {
	t.Log(GetUnixUS())
}

func TestGetUnixNS(t *testing.T) {
	t.Log(GetUnixNS())
}

func TestGetOldZeroTime(t *testing.T) {
	t.Log(GetOldZeroTime(0, 1))
}

func TestGetOldTime(t *testing.T) {
	t.Log(GetOldTime(1, 0))
}
