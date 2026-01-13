#!/bin/bash
set -e

echo ""
echo -e "\033[0;36m================================"
echo -e "   TUNNELCOW BUILD SCRIPT"
echo -e "================================\033[0m"
echo ""


DATA_PATH="data"
mkdir -p "$DATA_PATH"
BUILD_FILE="$DATA_PATH/.build_info.json"

if [ ! -f "$BUILD_FILE" ]; then
    echo '{"total": 0, "patch": 0}' > "$BUILD_FILE"
fi



TOTAL=$(grep -o '"total": *[0-9]*' "$BUILD_FILE" | cut -d: -f2 | tr -d ' ')
PATCH=$(grep -o '"patch": *[0-9]*' "$BUILD_FILE" | cut -d: -f2 | tr -d ' ')


TOTAL=$((TOTAL + 1))
VERSION="0.$TOTAL.$PATCH"


echo "{\"total\": $TOTAL, \"patch\": $PATCH}" > "$BUILD_FILE"

echo -e "\033[0;35m[1/6] Build Configuration\033[0m"
echo -e "      Version: \033[1;37mv$VERSION\033[0m"
echo -e "      Build: \033[0;37m#$TOTAL\033[0m"
echo ""


echo -e "\033[0;35m[2/6] Updating App.jsx Version...\033[0m"
APP_PATH="web/src/App.jsx"

if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "s/v[0-9]*\.[0-9]*\.[0-9]*/v$VERSION/" "$APP_PATH"
else
    sed -i "s/v[0-9]*\.[0-9]*\.[0-9]*/v$VERSION/" "$APP_PATH"
fi
echo -e "\033[0;32m      [OK] Updated to v$VERSION\033[0m"
echo ""


echo -e "\033[0;35m[3/6] Building Frontend...\033[0m"
cd web
npm run build
cd ..
echo -e "\033[0;32m      [OK] Frontend build complete\033[0m"
echo ""


echo -e "\033[0;35m[4/6] Embedding Static Assets...\033[0m"
DEST="cmd/tunnelcow-client/web_dist"
rm -rf "$DEST"
mkdir -p "$DEST"
cp -r web/dist/* "$DEST"
FILE_COUNT=$(find "$DEST" -type f | wc -l)
echo -e "\033[0;32m      [OK] Embedded $FILE_COUNT files\033[0m"
echo ""


echo -e "\033[0;35m[5/6] Compiling Go Binaries...\033[0m"


echo -e "\033[0;37m      Building Linux Client...\033[0m"
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$VERSION -s -w" -o tunnelcow-client-linux ./cmd/tunnelcow-client
echo -e "\033[0;32m      [OK] Client (Linux)\033[0m"


echo -e "\033[0;37m      Building Linux Server...\033[0m"
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$VERSION -s -w" -o tunnelcow-server-linux ./cmd/tunnelcow-server
echo -e "\033[0;32m      [OK] Server (Linux)\033[0m"


echo -e "\033[0;37m      Building Windows Binaries...\033[0m"
GOOS=windows GOARCH=amd64 go build -ldflags "-X main.Version=$VERSION -s -w" -o tunnelcow-client.exe ./cmd/tunnelcow-client
GOOS=windows GOARCH=amd64 go build -ldflags "-X main.Version=$VERSION -s -w" -o tunnelcow-server.exe ./cmd/tunnelcow-server
echo -e "\033[0;32m      [OK] Windows Binaries\033[0m"
echo ""


echo -e "\033[0;32m================================"
echo -e "  BUILD COMPLETE"
echo -e "================================\033[0m"
echo ""
echo -e "  Version: \033[1;37mv$VERSION\033[0m"
echo -e "  Build:   \033[0;37m#$TOTAL\033[0m"
echo ""
echo -e "  Artifacts:"
echo -e "    - tunnelcow-client.exe"
echo -e "    - tunnelcow-server.exe"
echo -e "    - tunnelcow-client-linux"
echo -e "    - tunnelcow-server-linux"
echo ""
echo -e "\033[0;36m  Ready to deploy!\033[0m"
echo ""
