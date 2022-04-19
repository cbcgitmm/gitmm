package rules

import (
	"github.com/zricethezav/gitleaks/v8/config"
)

func MailGunPrivateAPIToken() *config.Rule {
	// define rule
	r := config.Rule{
		RuleID:      "mailgun-private-api-token",
		Description: "Mailgun private API token",
		Regex:       generateSemiGenericRegex([]string{"mailgun"}, `key-[a-f0-9]{32}`),
		SecretGroup: 1,
		Keywords: []string{
			"mailgun",
		},
	}

	// validate
	tps := []string{
		generateSampleSecret("mailgun", "key-"+sampleHex32Token),
	}
	return validate(r, tps)
}

func MailGunPubAPIToken() *config.Rule {
	// define rule
	r := config.Rule{
		RuleID:      "mailgun-pub-key",
		Description: "Mailgun public validation key",
		Regex:       generateSemiGenericRegex([]string{"mailgun"}, `pubkey-[a-f0-9]{32}`),
		SecretGroup: 1,
		Keywords: []string{
			"mailgun",
		},
	}

	// validate
	tps := []string{
		generateSampleSecret("mailgun", "pubkey-"+sampleHex32Token),
	}
	return validate(r, tps)
}

func MailGunSigningKey() *config.Rule {
	// define rule
	r := config.Rule{
		RuleID:      "mailgun-signing-key",
		Description: "Mailgun webhook signing key",
		Regex:       generateSemiGenericRegex([]string{"mailgun"}, `[a-h0-9]{32}-[a-h0-9]{8}-[a-h0-9]{8}`),
		SecretGroup: 1,
		Keywords: []string{
			"mailgun",
		},
	}

	// validate
	tps := []string{
		generateSampleSecret("mailgun", sampleHex32Token+"-00001111-22223333"),
	}
	return validate(r, tps)
}
