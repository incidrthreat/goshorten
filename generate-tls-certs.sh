#!/bin/bash

# This script generates the private and public keys for TLS secure communications
# and moves the files to the appropriate directories.

# Variables, Change as necessary.
C="US"
ST="Some State"
L="Docker"
O="GoShortener"
OU="OpenSourceStuffs"

# Prompt user to enter password twice
read -s -p "Password: " password
echo 
read -s -p "Password (again): " password2

# Check if passwords match and if not ask again
while [ "$password" != "$password2" ];
do
    echo 
    echo "Please try again"
    read -s -p "Password: " password
    echo
    read -s -p "Password (again): " password2
done
echo

# Clear password2 variable
unset password2

# Generate private KEY
openssl genrsa -out server.key 2048

# Generate CSR
# DO NOT change the CN variable in "-subj"
openssl req -new -sha256 -key server.key -out server.csr -subj "/C=$C/ST=$ST/L=$L/O=$O/OU=$OU/CN=grpcbackend" -passin "pass:$password"

# Clear password variable
unset password

# Create CRT
openssl x509 -req -days 3650 -in server.csr -out server.crt -signkey server.key

# Move files to proper places.
cp server.crt ./frontend-go/
mv server.* ./backend/

# Inform user script is done.
echo "All files created and moved into the proper directories.  Proceed with 'docker-compose up'"