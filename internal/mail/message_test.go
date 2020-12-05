package mail

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zostay/dotfiles-go/internal/secrets"
)

const (
	badEmail = `Delivered-To: sterling@example.com
Received: by 10.0.0.1 with SMTP id abc123;
        Tue, 28 May 2013 12:39:19 -0700 (PDT)
X-Received: by 10.0.0.1 with SMTP id abc123;
        Tue, 28 May 2013 12:39:19 -0700 (PDT)
Return-Path: <support@example.com.com>
Received: from cd2.example.com (mx1.example.com. [10.0.0.1])
        by mx.example.com with ESMTP id abc123
        for <sterling@example.com>;
        Tue, 28 May 2013 12:39:19 -0700 (PDT)
Received-SPF: neutral (example.com: 10.0.0.1 is neither permitted nor denied by best guess record for domain of support@example.com) client-ip=10.0.0.1;
Authentication-Results: mx.example.com;
       spf=neutral (example.com: 10.0.0.1 is neither permitted nor denied by best guess record for domain of support@example.com) smtp.mail=support@example.com
Received: from WEB03.example.com (web03.example.com [10.0.0.1] (may be forged))
	by cd2.example.com (8.13.8/8.13.8) with ESMTP id abc123
	for <sterling@example.com>; Tue, 28 May 2013 14:39:18 -0500
Received: from mail pickup service by WEB03.example.com with Microsoft SMTPSVC;
	 Tue, 28 May 2013 14:38:59 -0500
To: sterling@example.com
From: support@example.com
Reply-To: support@example.com
Subject: Example Registration
MIME-Version: 1.0
Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: 8-bit
Message-ID: <WEB03@WEB03.example.com>
X-OriginalArrivalTime: 28 May 2013 19:38:59.0062 (UTC) FILETIME=[FFA98160:01CE5BDA]
Date: 28 May 2013 14:38:59 -0500
Keywords: Account Info \Important

Thank you for registering with Example.

Your login information is below.  You can access your profile at: https://www.example.com
Username: example
Password: secret

Thank you,
Example, Inc.
`
)

func TestFixHeadersReader(t *testing.T) {
	t.Parallel()

	secrets.QuickSetKeepers(secrets.MustNewInternal())

	sr := strings.NewReader(badEmail)
	fr, err := fixHeadersReader(sr)
	assert.NoError(t, err, "no error from constructing fixHeadersReader()")

	bs, err := ioutil.ReadAll(fr)
	assert.NoError(t, err, "no error from reading fixHeadersReader() buffer")

	s := string(bs)
	assert.NotContains(t, s, "\nContent-Transfer-Encoding: 8-bit\n", "offensive CTE is removed")
	assert.Contains(t, s, "\nContent-Transfer-Encoding: 8bit\n", "correct CTE is present")
}
