# TOTP CLI

A command-line tool for managing TOTP (Time-Based One-Time Password) entries. This tool allows you to add, 
ist, generate, and remove TOTP entries. It stores the entries in file with AES encryption.

## Features

- Add TOTP entries using OTP URLs
- Add TOPT enries using QRC image file
- List all stored TOTP entries
- Generate TOTP codes for specified accounts
- Remove TOTP entries by account and issuer
- Option to specify the database file path via a 
  command-line argument or environment variable


## How to Use the TOTP CLI Application

### 1. Installation

1. **Clone the Repository:**

   ```bash
   git clone <repository_url>
   ```

2. **Build the Application:**

```bash
   cd totp
   go build -o totp ./cmd/...
   # or 
   make rotp
```

3. **Run the Application:**

   ```bash
   ./totp
   ```

### 2. Commands

#### Create a New Database

To create a new TOTP database file, run:

```bash
./totp create-db
#or
./totp c
#or
./totp db
```

#### Add a TOTP from URL

To add a new TOTP using a URL, run:
```bash
./totp add-url -u "otpauth://totp/Issuer:AccountName?secret=YOUR_SECRET_KEY&issuer=Issuer&digits=6&algorithm=SHA1&period=30"
# or 
./totp a -u "otpauth://totp/Issuer:AccountName?secret=YOUR_SECRET_KEY&issuer=Issuer&digits=6&algorithm=SHA1&period=30"
```

#### Add a TOTP from QR Code

To add a new TOTP by scanning a QR code from an image file, run:
```bash
./totp add-qrc -i path/to/image.png
# or
./totp grc -i path/to/image.png
```

#### List All TOTPs

To list all TOTPs stored in the database, run:
```bash
./totp list
# or
./totp l
```

#### Generate a TOTP

To generate a TOTP for a specific account and issuer, run:
```bash
./totp generate -a AccountName -i IssuerName
# or
./totp g -a AccountName -i IssuerName
# or
./totp gen -a AccountName
```
You may specify only AccountName if i's uniqe.

#### Remove a TOTP

To remove a TOTP for a specific account and issuer, run:
```bash
./totp remove -a AccountName -i IssuerName
# or
./totp rm -a AccountName -i IssuerName
```


### 3. Flags

- `-d, --db`: Path to the database file.
- `-s, --salt`: Salt input for encryption. If you want to use you own one  but default.
- `-q, --quiet`: Suppress output.

### 4. Environment Variables

- `TOTP_DB_PATH`: Path to the database file. Overrides the by -d flag.
- `TOTP_SALT`: Salt input for encryption. Overrides the by -s flag.

### 4. Contributing

If you find any issues or have suggestions for improvements, feel free to open an issue or submit a pull request.

### 5. License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
