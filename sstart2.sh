#!/bin/bash
source .env
set -e
echo "Starting the script"
export PATH=$path
export FABRIC_CFG_PATH=$FABRIC_CFG_PATH
export CORE_PEER_LOCALMSPID=$CORE_PEER_LOCALMSPID
export CORE_PEER_TLS_ENABLED=$CORE_PEER_TLS_ENABLED
export CORE_PEER_TLS_CLIENTAUTHREQUIRED=$CORE_PEER_TLS_CLIENTAUTHREQUIRED
export CORE_PEER_TLS_ROOTCERT_FILE=$CORE_PEER_TLS_ROOTCERT_FILE
export CORE_PEER_TLS_CLIENTCERT_FILE=$CORE_PEER_TLS_CLIENTCERT_FILE
export CORE_PEER_TLS_CLIENTKEY_FILE=$CORE_PEER_TLS_CLIENTKEY_FILE
export CORE_PEER_MSPCONFIGPATH=$CORE_PEER_MSPCONFIGPATH
export CHANNEL_NAME=$CHANNEL_NAME
export CC_NAME=$CC_NAME
export CC_VERSION="11x11.00"
export CC_SEQUENCE="11x11"
export CC_PATH=$CC_PATH
export CC_PACKAGE=$CC_PACKAGE"_11x11.0.tar.gz"
export CC_LABEL=$CC_LABEL"_11x11.0"
export ORDERER_ADDRESS="localhost:7050"
export CORE_PEER_ADDRESS="localhost:7011"
export CORE_PEER_CHAINCODELISTENADDRESS="0.0.0.0:7021"
export CORE_PEER_CHAINCODEADDRESS="localhost:7021"
export CC_PACKAGE_ID=22x22
peer lifecycle chaincode approveformyorg -o $ORDERER_ADDRESS --channelID $CHANNEL_NAME --name $CC_NAME --version $CC_VERSION --package-id $CC_PACKAGE_ID --sequence $CC_SEQUENCE --tls --cafile $CORE_PEER_TLS_ROOTCERT_FILE
peer lifecycle chaincode checkcommitreadiness --channelID $CHANNEL_NAME --name $CC_NAME --version $CC_VERSION --sequence $CC_SEQUENCE --tls --cafile $CORE_PEER_TLS_ROOTCERT_FILE --output json
peer lifecycle chaincode commit -o $ORDERER_ADDRESS --channelID $CHANNEL_NAME --name $CC_NAME --version $CC_VERSION --sequence $CC_SEQUENCE --tls --cafile $CORE_PEER_TLS_ROOTCERT_FILE --peerAddresses $CORE_PEER_ADDRESS --tlsRootCertFiles $CORE_PEER_TLS_ROOTCERT_FILE
peer lifecycle chaincode querycommitted --channelID $CHANNEL_NAME --name $CC_NAME --cafile $CORE_PEER_TLS_ROOTCERT_FILE
echo "Script ended"
