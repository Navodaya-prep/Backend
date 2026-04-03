# Email Setup Guide for Admin Invitations

## Overview

When you invite a new admin, the system generates a temporary password that needs to be sent via email to the new admin. Currently, the password is shown in the UI, but for production, you should implement automated email sending.

## Email Service Options

### Option 1: Gmail SMTP (Easiest for testing)

**Pros:** Free, easy to set up for testing
**Cons:** Rate limited, not suitable for production

1. **Enable 2-Factor Authentication** on your Gmail account
2. **Create an App Password:**
   - Go to Google Account settings → Security
   - Enable 2-Step Verification
   - Go to "App passwords"
   - Generate a new app password for "Mail"

3. **Add to .env file:**
```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
```

### Option 2: SendGrid (Recommended for Production)

**Pros:** Free tier (100 emails/day), reliable, easy API
**Cons:** Requires email verification

1. **Sign up** at https://sendgrid.com/
2. **Create an API Key** in Settings → API Keys
3. **Verify your sender email** in Settings → Sender Authentication

4. **Add to .env file:**
```bash
SENDGRID_API_KEY=your-api-key-here
SENDGRID_FROM_EMAIL=noreply@yourdomain.com
SENDGRID_FROM_NAME=Navodaya Admin
```

### Option 3: AWS SES (Best for Scale)

**Pros:** Very cheap, scalable, reliable
**Cons:** Requires AWS account, more complex setup

1. **Set up AWS SES** in AWS Console
2. **Verify your domain** or email address
3. **Create IAM credentials** with SES permissions

4. **Add to .env file:**
```bash
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your-access-key
AWS_SECRET_ACCESS_KEY=your-secret-key
AWS_SES_FROM_EMAIL=noreply@yourdomain.com
```

## Implementation Guide

### Step 1: Choose Your Email Service

I recommend **SendGrid for production** and **Gmail SMTP for testing**.

### Step 2: Install Email Package in Go

```bash
cd navodaya-go
go get github.com/sendgrid/sendgrid-go
```

Or for SMTP:
```bash
go get gopkg.in/gomail.v2
```

### Step 3: Create Email Utility

I'll create a ready-to-use email utility file for you:

**For SendGrid:**

Create `navodaya-go/utils/email.go`:

```go
package utils

import (
	"fmt"
	"os"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

func SendAdminInviteEmail(toEmail, firstName, lastName, tempPassword string) error {
	from := mail.NewEmail(
		os.Getenv("SENDGRID_FROM_NAME"),
		os.Getenv("SENDGRID_FROM_EMAIL"),
	)
	to := mail.NewEmail(fmt.Sprintf("%s %s", firstName, lastName), toEmail)
	
	subject := "Welcome to Navodaya Admin Panel"
	
	htmlContent := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
			<div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 40px 20px; text-align: center;">
				<h1 style="color: white; margin: 0;">🏫 Navodaya Prime</h1>
			</div>
			<div style="padding: 40px 20px;">
				<h2>Welcome, %s!</h2>
				<p>You've been invited to join the Navodaya Prime Admin Panel.</p>
				
				<div style="background: #f3f4f6; padding: 20px; border-radius: 8px; margin: 20px 0;">
					<p style="margin: 0 0 10px 0;"><strong>Your Login Credentials:</strong></p>
					<p style="margin: 5px 0;"><strong>Email:</strong> %s</p>
					<p style="margin: 5px 0;"><strong>Temporary Password:</strong> <code style="background: white; padding: 5px 10px; border-radius: 4px; font-size: 16px;">%s</code></p>
				</div>
				
				<p>⚠️ <strong>Important:</strong> Please change your password immediately after your first login.</p>
				
				<div style="text-align: center; margin: 30px 0;">
					<a href="https://admin.navodayaprime.com" style="background: #667eea; color: white; padding: 12px 30px; text-decoration: none; border-radius: 6px; display: inline-block;">
						Login to Admin Panel
					</a>
				</div>
				
				<hr style="border: none; border-top: 1px solid #e5e7eb; margin: 30px 0;">
				
				<p style="color: #6b7280; font-size: 14px;">
					If you have any questions, please contact your super admin.
				</p>
			</div>
		</body>
		</html>
	`, firstName, toEmail, tempPassword)
	
	plainTextContent := fmt.Sprintf(`
Welcome to Navodaya Prime Admin Panel, %s!

Your Login Credentials:
Email: %s
Temporary Password: %s

Important: Please change your password immediately after your first login.

Login at: https://admin.navodayaprime.com

If you have any questions, please contact your super admin.
	`, firstName, toEmail, tempPassword)
	
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	response, err := client.Send(message)
	
	if err != nil {
		return err
	}
	
	if response.StatusCode >= 400 {
		return fmt.Errorf("sendgrid error: %d", response.StatusCode)
	}
	
	return nil
}
```

**For Gmail SMTP:**

```go
package utils

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

func SendAdminInviteEmail(toEmail, firstName, lastName, tempPassword string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", os.Getenv("SMTP_USER"))
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "Welcome to Navodaya Admin Panel")
	
	htmlBody := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif;">
			<h2>Welcome, %s!</h2>
			<p>You've been invited to join the Navodaya Prime Admin Panel.</p>
			<p><strong>Email:</strong> %s</p>
			<p><strong>Temporary Password:</strong> <code>%s</code></p>
			<p>⚠️ Please change your password after first login.</p>
		</body>
		</html>
	`, firstName, toEmail, tempPassword)
	
	m.SetBody("text/html", htmlBody)
	
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	d := gomail.NewDialer(
		os.Getenv("SMTP_HOST"),
		port,
		os.Getenv("SMTP_USER"),
		os.Getenv("SMTP_PASSWORD"),
	)
	
	return d.DialAndSend(m)
}
```

### Step 4: Update the Invite Handler

Modify `navodaya-go/handlers/admin_management.go`:

```go
// After creating the admin, send email
err = utils.SendAdminInviteEmail(
	newAdmin.Email,
	newAdmin.FirstName,
	newAdmin.LastName,
	tempPassword,
)

if err != nil {
	// Log the error but don't fail the invitation
	fmt.Printf("Failed to send email: %v\n", err)
	// Still return the password in the response as fallback
	utils.Success(c, http.StatusCreated, gin.H{
		"admin":        newAdmin,
		"tempPassword": tempPassword,
		"emailSent":    false,
		"message":      "Admin created but email failed to send. Please send the password manually.",
	}, "Admin invited (email failed)")
	return
}

// Email sent successfully
utils.Success(c, http.StatusCreated, gin.H{
	"admin":     newAdmin,
	"emailSent": true,
	"message":   "Admin invited successfully. Login credentials sent via email.",
}, "Admin invited successfully")
```

### Step 5: Update .env File

Add your email credentials to the `.env` file in the root of navodaya-go:

```bash
# SendGrid (Recommended)
SENDGRID_API_KEY=SG.xxxxxxxxxxxxxxxxxxxxxxxx
SENDGRID_FROM_EMAIL=noreply@navodayaprime.com
SENDGRID_FROM_NAME=Navodaya Admin

# OR Gmail SMTP (For Testing)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
```

## Testing Your Email Setup

1. **Start your backend** with the environment variables set
2. **Invite a test admin** through the admin panel
3. **Check the recipient's inbox** (and spam folder)
4. **Verify the email content** looks correct

## Troubleshooting

### Email Not Sending (Gmail)
- Verify 2FA is enabled
- Check App Password is correct (not your regular password)
- Check Gmail's "Less secure app access" settings

### Email Going to Spam
- Use a verified domain email address
- Add SPF and DKIM records to your domain
- Avoid spam trigger words in subject/body
- Consider using a professional email service like SendGrid

### SendGrid Errors
- Verify your API key is correct
- Check sender email is verified
- Ensure you're not exceeding free tier limits (100/day)
- Check SendGrid dashboard for error logs

## Security Best Practices

1. **Never commit credentials** - Always use environment variables
2. **Use strong API keys** - Rotate them regularly
3. **Rate limit invitations** - Prevent spam/abuse
4. **Log email attempts** - Track delivery status
5. **Validate email addresses** - Before sending

## Production Checklist

- [ ] Choose production email service (SendGrid recommended)
- [ ] Set up domain authentication (SPF, DKIM, DMARC)
- [ ] Configure environment variables in production
- [ ] Test email delivery to multiple providers (Gmail, Outlook, etc.)
- [ ] Set up email monitoring/alerts
- [ ] Create professional email templates
- [ ] Add unsubscribe link (if sending marketing emails)
- [ ] Comply with email regulations (CAN-SPAM, GDPR)

## Cost Estimates

| Service | Free Tier | Paid Plans |
|---------|-----------|------------|
| SendGrid | 100 emails/day | $14.95/month (40k emails) |
| AWS SES | 62,000 emails/month (first year) | $0.10/1000 emails |
| Gmail SMTP | ~500/day (unofficial) | Not recommended for production |

## Next Steps

1. Choose your email service
2. Set up credentials
3. Install the Go package
4. Create the email utility file
5. Update the invite handler
6. Test with a real email
7. Deploy to production

Need help with any of these steps? Let me know!
