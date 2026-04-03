# Admin Setup Guide

## Creating the First Super Admin

Since the admin login now requires email and password authentication, you need to create the first super admin user in the database.

### Option 1: Using MongoDB Shell

```bash
# Connect to your MongoDB database
mongosh "your-mongodb-connection-string"

# Switch to your database (usually 'navodaya')
use navodaya

# Insert the first super admin
# Note: The password below is 'admin123' hashed with bcrypt (cost 10)
db.admins.insertOne({
  name: "Super Admin",
  email: "admin@navodaya.com",
  password: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
  isSuperAdmin: true,
  isActive: true,
  createdAt: new Date(),
  updatedAt: new Date()
})
```

### Option 2: Using the Seed Script

Run the seed script to create the default super admin:

```bash
cd navodaya-go
go run scripts/seed_admin.go
```

## Default Login Credentials

After running either option above, you can login with:

- **Email:** admin@navodaya.com
- **Password:** admin123

⚠️ **IMPORTANT:** Change this password immediately after first login!

## Creating Additional Admins

Once logged in as a super admin, you can:

1. Access the admin profile page
2. Create new admin accounts (if UI is implemented)
3. Set `isSuperAdmin: false` for regular admins
4. Set `isSuperAdmin: true` for additional super admins

## Admin Roles

- **Super Admin** (`isSuperAdmin: true`): Can manage other admins, has full access
- **Regular Admin** (`isSuperAdmin: false`): Can manage content but not other admins

## Password Security

- All passwords are hashed using bcrypt with cost factor 10
- Passwords are never stored in plain text
- Passwords are hidden in API responses (`json:"-"` tag)

## Changing an Admin Password

### Via API (when logged in)

```bash
curl -X PUT http://localhost:8080/api/admin/auth/password \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "currentPassword": "admin123",
    "newPassword": "your-new-secure-password"
  }'
```

### Via MongoDB (if locked out)

```bash
mongosh "your-mongodb-connection-string"

use navodaya

# Generate a bcrypt hash of your new password at: https://bcrypt-generator.com/
# Use cost factor 10

db.admins.updateOne(
  { email: "admin@navodaya.com" },
  { $set: { 
    password: "YOUR_BCRYPT_HASH_HERE",
    updatedAt: new Date()
  }}
)
```

## API Endpoints

### Public Endpoints

- `POST /api/admin/auth/login` - Login with email and password

### Protected Endpoints (Require JWT Token)

- `GET /api/admin/auth/profile` - Get current admin profile
- `PUT /api/admin/auth/profile` - Update admin profile (name, email)
- `PUT /api/admin/auth/password` - Change password

All other admin endpoints require the `Authorization: Bearer <token>` header.

## Troubleshooting

### "Invalid admin token" error

- Your JWT token may have expired (tokens last 7 days)
- Login again to get a new token

### "Unauthorized" error

- Missing or invalid Authorization header
- Ensure you're sending: `Authorization: Bearer YOUR_TOKEN_HERE`

### Can't login

- Check email and password are correct
- Verify admin exists in database: `db.admins.findOne({ email: "admin@navodaya.com" })`
- Verify `isActive` is `true`
- Check backend logs for detailed error messages
