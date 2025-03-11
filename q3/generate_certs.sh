#!/bin/bash

# Exit immediately if any command fails
set -e

# Create a directory for certificates
CERTS_DIR="certs"
mkdir -p $CERTS_DIR

echo "🔹 Generating TLS Certificates in $CERTS_DIR..."

# ===========================
# 1️⃣ Generate CA Certificate
# ===========================
echo "🔹 Creating Certificate Authority (CA)..."
openssl genrsa -out $CERTS_DIR/ca.key 4096
openssl req -x509 -new -nodes -key $CERTS_DIR/ca.key -sha256 -days 365 -out $CERTS_DIR/ca.crt -subj "/CN=MyCA"

# ===========================
# 2️⃣ Generate Payment Gateway Server Certificate
# ===========================
echo "🔹 Creating Payment Gateway Server Certificate..."
openssl genrsa -out $CERTS_DIR/payment_gateway.key 4096
openssl req -new -key $CERTS_DIR/payment_gateway.key -out $CERTS_DIR/payment_gateway.csr -subj "/CN=payment_gateway"

# Create a config file for SAN (Subject Alternative Name)
cat <<EOF > $CERTS_DIR/payment_gateway.ext
subjectAltName = DNS:payment_gateway, DNS:localhost, IP:127.0.0.1
EOF

# Sign the Payment Gateway server certificate with the CA
openssl x509 -req -in $CERTS_DIR/payment_gateway.csr -CA $CERTS_DIR/ca.crt -CAkey $CERTS_DIR/ca.key -CAcreateserial \
    -out $CERTS_DIR/payment_gateway.crt -days 365 -sha256 -extfile $CERTS_DIR/payment_gateway.ext

# ===========================
# 3️⃣ Generate Bank Server Certificate
# ===========================
echo "🔹 Creating Bank Server Certificate..."
openssl genrsa -out $CERTS_DIR/bank_server.key 4096
openssl req -new -key $CERTS_DIR/bank_server.key -out $CERTS_DIR/bank_server.csr -subj "/CN=bank_server"

# Create a config file for SAN (Subject Alternative Name)
cat <<EOF > $CERTS_DIR/bank_server.ext
subjectAltName = DNS:bank_server, DNS:localhost, IP:127.0.0.1
EOF

# Sign the Bank Server certificate with the CA
openssl x509 -req -in $CERTS_DIR/bank_server.csr -CA $CERTS_DIR/ca.crt -CAkey $CERTS_DIR/ca.key -CAcreateserial \
    -out $CERTS_DIR/bank_server.crt -days 365 -sha256 -extfile $CERTS_DIR/bank_server.ext

# ===========================
# 4️⃣ Generate Client Certificate
# ===========================
echo "🔹 Creating Client Certificate..."
openssl genrsa -out $CERTS_DIR/client.key 4096
openssl req -new -key $CERTS_DIR/client.key -out $CERTS_DIR/client.csr -subj "/CN=client"

# Sign the client certificate with the CA
openssl x509 -req -in $CERTS_DIR/client.csr -CA $CERTS_DIR/ca.crt -CAkey $CERTS_DIR/ca.key -CAcreateserial \
    -out $CERTS_DIR/client.crt -days 365 -sha256

# ===========================
# 5️⃣ Clean Up and Display Results
# ===========================
echo "✅ Certificates Generated Successfully!"
ls -l $CERTS_DIR

echo "🔹 Files Created:"
echo "- CA Certificate: $CERTS_DIR/ca.crt"
echo "- Payment Gateway Certificate: $CERTS_DIR/payment_gateway.crt"
echo "- Payment Gateway Key: $CERTS_DIR/payment_gateway.key"
echo "- Bank Server Certificate: $CERTS_DIR/bank_server.crt"
echo "- Bank Server Key: $CERTS_DIR/bank_server.key"
echo "- Client Certificate: $CERTS_DIR/client.crt"
echo "- Client Key: $CERTS_DIR/client.key"
echo "- CA Key (Keep Secret!): $CERTS_DIR/ca.key"
# #!/bin/bash

# # Exit immediately if any command fails
# set -e

# # Create a directory for certificates
# CERTS_DIR="certs"
# mkdir -p $CERTS_DIR

# echo "🔹 Generating TLS Certificates in $CERTS_DIR..."

# # ===========================
# # 1️⃣ Generate CA Certificate
# # ===========================
# echo "🔹 Creating Certificate Authority (CA)..."
# openssl genrsa -out $CERTS_DIR/ca.key 4096
# openssl req -x509 -new -nodes -key $CERTS_DIR/ca.key -sha256 -days 365 -out $CERTS_DIR/ca.crt -subj "/CN=MyCA"

# # ===========================
# # 2️⃣ Generate Server Certificate
# # ===========================
# echo "🔹 Creating Server Certificate..."
# openssl genrsa -out $CERTS_DIR/payment_gateway.key 4096
# openssl req -new -key $CERTS_DIR/payment_gateway.key -out $CERTS_DIR/payment_gateway.csr -subj "/CN=localhost"

# # Create a config file for SAN (Subject Alternative Name)
# cat <<EOF > $CERTS_DIR/payment_gateway.ext
# subjectAltName = DNS:localhost
# EOF

# # Sign the server certificate with the CA
# openssl x509 -req -in $CERTS_DIR/payment_gateway.csr -CA $CERTS_DIR/ca.crt -CAkey $CERTS_DIR/ca.key -CAcreateserial \
#     -out $CERTS_DIR/payment_gateway.crt -days 365 -sha256 -extfile $CERTS_DIR/payment_gateway.ext

# # ===========================
# # 3️⃣ Generate Client Certificate
# # ===========================
# echo "🔹 Creating Client Certificate..."
# openssl genrsa -out $CERTS_DIR/client.key 4096
# openssl req -new -key $CERTS_DIR/client.key -out $CERTS_DIR/client.csr -subj "/CN=client"

# # Sign the client certificate with the CA
# openssl x509 -req -in $CERTS_DIR/client.csr -CA $CERTS_DIR/ca.crt -CAkey $CERTS_DIR/ca.key -CAcreateserial \
#     -out $CERTS_DIR/client.crt -days 365 -sha256

# # ===========================
# # 4️⃣ Clean Up and Display Results
# # ===========================
# echo "✅ Certificates Generated Successfully!"
# ls -l $CERTS_DIR

# echo "🔹 Files Created:"
# echo "- CA Certificate: $CERTS_DIR/ca.crt"
# echo "- Server Certificate: $CERTS_DIR/payment_gateway.crt"
# echo "- Server Key: $CERTS_DIR/payment_gateway.key"
# echo "- Client Certificate: $CERTS_DIR/client.crt"
# echo "- Client Key: $CERTS_DIR/client.key"
# echo "- CA Key (Keep Secret!): $CERTS_DIR/ca.key"
