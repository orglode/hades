package crypto4go

import (
	"testing"
)

func TestParseCertificate(t *testing.T) {
	var k = "MIIE4jCCAsqgAwIBAgIIYsSr5bKAMl8wDQYJKoZIhvcNAQELBQAwejELMAkGA1UEBhMCQ04xFjAUBgNVBAoMDUFudCBGaW5hbmNpYWwxIDAeBgNVBAsMF0NlcnRpZmljYXRpb24gQXV0aG9yaXR5MTEwLwYDVQQDDChBbnQgRmluYW5jaWFsIENlcnRpZmljYXRpb24gQXV0aG9yaXR5IFIxMB4XDTE4MDMyMjE0MzQxNVoXDTM3MTEyNjE0MzQxNVowgYIxCzAJBgNVBAYTAkNOMRYwFAYDVQQKDA1BbnQgRmluYW5jaWFsMSAwHgYDVQQLDBdDZXJ0aWZpY2F0aW9uIEF1dGhvcml0eTE5MDcGA1UEAwwwQW50IEZpbmFuY2lhbCBDZXJ0aWZpY2F0aW9uIEF1dGhvcml0eSBDbGFzcyAyIFIxMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAsLMfYaoRoPRbmDcAfXPCmKf43pWRN5yTXa/KJWO0l+mrgQvs89bANEvbDUxlkGwycwtwi5DgBuBgVhLliXu+R9CYgr2dXs8D8Hx/gsggDcyGPLmVrDOnL+dyeauheARZfA3du60fwEwwbGcVIpIxPa/4n3IS/ElxQa6DNgqxh8J9Xwh7qMGl0JK9+bALuxf7B541Gr4p0WENG8fhgjBV4w4ut9eQLOoa1eddOUSZcy46Z7allwowwgt7b5VFfx/P1iKJ3LzBMgkCK7GZ2kiLrL7RiqV+h482J7hkJD+ardoc6LnrHO/hIZymDxok+VH9fVeUdQa29IZKrIDVj65THQIDAQABo2MwYTAfBgNVHSMEGDAWgBRfdLQEwE8HWurlsdsio4dBspzhATAdBgNVHQ4EFgQUSqHkYINtUSAtDPnS8XoyoP9p7qEwDwYDVR0TAQH/BAUwAwEB/zAOBgNVHQ8BAf8EBAMCAQYwDQYJKoZIhvcNAQELBQADggIBAIQ8TzFy4bVIVb8+WhHKCkKNPcJe2EZuIcqvRoi727lZTJOfYy/JzLtckyZYfEI8J0lasZ29wkTta1IjSo+a6XdhudU4ONVBrL70U8Kzntplw/6TBNbLFpp7taRALjUgbCOk4EoBMbeCL0GiYYsTS0mw7xdySzmGQku4GTyqutIGPQwKxSj9iSFw1FCZqr4VP4tyXzMUgc52SzagA6i7AyLedd3tbS6lnR5BL+W9Kx9hwT8L7WANAxQzv/jGldeuSLN8bsTxlOYlsdjmIGu/C9OWblPYGpjQQIRyvs4Cc/mNhrh+14EQgwuemIIFDLOgcD+iISoN8CqegelNcJndFw1PDN6LkVoiHz9p7jzsge8RKay/QW6C03KNDpWZEUCgCUdfHfo8xKeR+LL1cfn24HKJmZt8L/aeRZwZ1jwePXFRVtiXELvgJuM/tJDIFj2KD337iV64fWcKQ/ydDVGqfDZAdcU4hQdsrPWENwPTQPfVPq2NNLMyIH9+WKx9Ed6/WzeZmIy5ZWpX1TtTolo6OJXQFeItMAjHxW/ZSZTok5IS3FuRhExturaInnzjYpx50a6kS34c5+c8hYq7sAtZ/CNLZmBnBCFDaMQqT8xFZJ5uolUaSeXxg7JFY1QsYp5RKvj4SjFwCGKJ2+hPPe9UyyltxOidNtxjaknOCeBHytOr"

	var cert, err = ParseCertificate(FormatCertificate(k))
	if err != nil {
		t.Fatal(err)
	}

	t.Log(cert.Issuer.String())
}