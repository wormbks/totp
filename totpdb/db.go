package totpdb

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/fxamacker/cbor/v2"
	"github.com/olekukonko/tablewriter"
	"github.com/pquerna/otp"
)

var (
	ErrEntryExists   = errors.New("TOTP entry already exists")
	ErrEntryNotFound = errors.New("TOTP entry not found")
)

const defaultSize = 10

// TOTPData represents the structure of the CBOR file.
type TOTPData struct {
	Entries []TOTPEntry // map[name]url
}

// TOTPEntry represents a TOTP entry with all the necessary details.
type TOTPEntry struct {
	Issuer      string `cbor:"issuer"`
	AccountName string `cbor:"account_name"`
	Secret      string `cbor:"secret"`
	Type        string `cbor:"type"`
	Period      uint64 `cbor:"period"`
	Digits      int    `cbor:"digits"`
	Algorithm   string `cbor:"algorithm"`
	URL         string `cbor:"url"`
}

// ToTOTPEntry converts a Key to a TOTPEntry.
func FromOTPKey(k *otp.Key) TOTPEntry {
	return TOTPEntry{
		Issuer:      k.Issuer(),
		AccountName: k.AccountName(),
		Secret:      k.Secret(),
		Type:        k.Type(),
		Period:      k.Period(),
		Digits:      int(k.Digits()),
		Algorithm:   k.Algorithm().String(),
		URL:         k.URL(),
	}
}

// AddEntry adds a new TOTP entry to TOTPData.
func (data *TOTPData) AddEntry(key *otp.Key) error {
	ent := FromOTPKey(key)

	if data.Entries == nil {
		data.Entries = make([]TOTPEntry, 0, defaultSize)
	}
	// Check if entry already exists
	_, err := data.FindEntry(ent.AccountName, ent.Issuer)

	if err != nil { // Entry doesn't exist
		data.Entries = append(data.Entries, ent)
		return nil
	}

	return ErrEntryExists
}

// FindEntry finds a TOTP entry in TOTPData.
func (data *TOTPData) FindEntry(name, issuer string) (int, error) {
	for ind, entry := range data.Entries {
		if entry.AccountName == name {
			if issuer == "" {
				return ind, nil
			} else if entry.Issuer == issuer {
				return ind, nil
			}
		}
	}
	return -1, ErrEntryNotFound
}

// GetEntry retrieves a TOTP entry from TOTPData.
func (data *TOTPData) GetEntry(name, issuer string) (TOTPEntry, error) {
	ind, err := data.FindEntry(name, issuer)
	if err != nil {
		return TOTPEntry{}, err
	}
	return data.Entries[ind], nil

}

// RemoveEntry removes a TOTP entry from TOTPData.
func (data *TOTPData) RemoveEntry(name string, issuer string) error {
	index, err := data.FindEntry(name, issuer)
	if err != nil {
		return err
	}
	data.Entries = append(data.Entries[:index], data.Entries[index+1:]...)
	return nil
}

// PrintTable prints all entries as a table.
func (data *TOTPData) PrintTable() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Issuer", "Account Name", "Type", "Period", "Digits", "Algorithm"})

	for _, ent := range data.Entries {
		table.Append([]string{
			ent.Issuer,
			ent.AccountName,
			ent.Type,
			fmt.Sprintf("%d", ent.Period),
			fmt.Sprintf("%d", ent.Digits),
			ent.Algorithm,
		})
	}

	table.Render()
}

// ReadCBORSec reads the encrypted CBOR data from the file, decrypts it, and unmarshals it into a TOTPData struct.
func ReadCBORSec(filename string, password string, salt []byte) (*TOTPData, error) {
	// Read the encrypted data from the file
	encryptedData, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Derive the encryption key
	key := DeriveKey([]byte(password), salt, 32)

	// Decrypt the data
	data, err := Decrypt(encryptedData, key)
	if err != nil {
		return nil, err
	}

	// Unmarshal the CBOR data
	var totpData TOTPData
	decoder := cbor.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&totpData); err != nil {
		return nil, err
	}

	return &totpData, nil
}

// WriteCBORSec marshals the TOTPData struct into CBOR, encrypts it, and writes it to the file.
func WriteCBORSec(filename string, data *TOTPData, password string, salt []byte) error {
	// Marshal the data into CBOR
	var buf bytes.Buffer
	encoder := cbor.NewEncoder(&buf)
	if err := encoder.Encode(data); err != nil {
		return err
	}

	// Derive the encryption key
	key := DeriveKey([]byte(password), salt, 32)

	// Encrypt the data
	encryptedData, err := Encrypt(buf.Bytes(), key)
	if err != nil {
		return err
	}

	// Write the encrypted data to the file
	return os.WriteFile(filename, encryptedData, 0644)
}

// ReadCBOR reads the TOTP data from a CBOR file.
func ReadCBOR(filename string) (*TOTPData, error) {
	data := &TOTPData{}

	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return data, err // Return empty data if file doesn't exist
		}
		return nil, err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	err = cbor.Unmarshal(bytes, data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// WriteCBOR writes the TOTP data to a CBOR file.
func WriteCBOR(filename string, data *TOTPData) error {
	bytes, err := cbor.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, bytes, 0644)
}
