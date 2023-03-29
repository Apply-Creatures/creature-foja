#!/usr/bin/env bash
# This is an update script for forgejo installed via the binary distribution
# from codeberg.org/forgejo/forgejo on linux as systemd service. It
# performs a backup and updates Forgejo in place.
# NOTE: This adds the GPG Signing Key of the Forgejo maintainers to the keyring.
# Depends on: bash, curl, xz, sha256sum. optionally jq, gpg
#   See section below for available environment vars.
#   When no version is specified, updates to the latest release.
# Examples:
#   upgrade.sh 1.15.10
#   forgejohome=/opt/forgejo forgejoconf=$forgejohome/app.ini upgrade.sh

# Check if forgejo service is running
if ! pidof forgejo &> /dev/null; then
  echo "Error: forgejo is not running."
  exit 1
fi

# Continue with rest of the script if forgejo is running
echo "Forgejo is running. Continuing with rest of script..."

# apply variables from environment
: "${forgejobin:="/usr/local/bin/forgejo"}"
: "${forgejohome:="/var/lib/forgejo"}"
: "${forgejoconf:="/etc/forgejo/app.ini"}"
: "${forgejouser:="git"}"
: "${sudocmd:="sudo"}"
: "${arch:="linux-amd64"}"
: "${service_start:="$sudocmd systemctl start forgejo"}"
: "${service_stop:="$sudocmd systemctl stop forgejo"}"
: "${service_status:="$sudocmd systemctl status forgejo"}"
: "${backupopts:=""}" # see `forgejo dump --help` for available options

function forgejocmd {
  if [[ $sudocmd = "su" ]]; then
    # `-c` only accept one string as argument.
    "$sudocmd" - "$forgejouser" -c "$(printf "%q " "$forgejobin" "--config" "$forgejoconf" "--work-path" "$forgejohome" "$@")"
  else
    "$sudocmd" --user "$forgejouser" "$forgejobin" --config "$forgejoconf" --work-path "$forgejohome" "$@"
  fi
}

function require {
  for exe in "$@"; do
    command -v "$exe" &>/dev/null || (echo "missing dependency '$exe'"; exit 1)
  done
}

# parse command line arguments
while true; do
  case "$1" in
    -v | --version ) forgejoversion="$2"; shift 2 ;;
    -y | --yes ) no_confirm="yes"; shift ;;
    --ignore-gpg) ignore_gpg="yes"; shift ;;
    "" | -- ) shift; break ;;
    * ) echo "Usage:  [<environment vars>] upgrade.sh [-v <version>] [-y] [--ignore-gpg]"; exit 1;; 
  esac
done

# exit once any command fails. this means that each step should be idempotent!
set -euo pipefail

if [[ -f /etc/os-release ]]; then
  os_release=$(cat /etc/os-release)

  if [[ "$os_release" =~ "OpenWrt" ]]; then
    sudocmd="su"
    service_start="/etc/init.d/forgejo start"
    service_stop="/etc/init.d/forgejo stop"
    service_status="/etc/init.d/forgejo status"
  else
    require systemctl
  fi
fi

require curl xz sha256sum "$sudocmd"

# select version to install
if [[ -z "${forgejoversion:-}" ]]; then
  require jq
  forgejoversion=$(curl --connect-timeout 10 -sL 'https://codeberg.org/api/v1/repos/forgejo/forgejo/releases?draft=false&pre-release=false&limit=1' -H 'accept: application/json' | jq -r '.[0].tag_name | sub("v"; "")')
  echo "Latest available version is $forgejoversion"
fi

# confirm update
echo "Checking currently installed version..."
current=$(forgejocmd --version | cut -d ' ' -f 3)
[[ "$current" == "$forgejoversion" ]] && echo "$current is already installed, stopping." && exit 1
if [[ -z "${no_confirm:-}"  ]]; then
  echo "Make sure to read the changelog first: https://codeberg.org/forgejo/forgejo/src/branch/forgejo/CHANGELOG.md"
  echo "Are you ready to update forgejo from ${current} to ${forgejoversion}? (y/N)"
  read -r confirm
  [[ "$confirm" == "y" ]] || [[ "$confirm" == "Y" ]] || exit 1
fi

echo "Upgrading forgejo from $current to $forgejoversion ..."

pushd "$(pwd)" &>/dev/null
cd "$forgejohome" # needed for forgejo dump later

# download new binary
binname="forgejo-${forgejoversion}-${arch}"
binurl="https://codeberg.org/forgejo/forgejo/releases/download/v${forgejoversion}/${binname}.xz"
echo "Downloading $binurl..."
curl --connect-timeout 10 --silent --show-error --fail --location -O "$binurl{,.sha256,.asc}"

# validate checksum & gpg signature
sha256sum -c "${binname}.xz.sha256"
if [[ -z "${ignore_gpg:-}" ]]; then
  require gpg
  gpg --keyserver keys.openpgp.org --recv EB114F5E6C0DC2BCDD183550A4B61A2DC5923710
  gpg --verify "${binname}.xz.asc" "${binname}.xz" || { echo 'Signature does not match'; exit 1; }
fi
rm "${binname}".xz.{sha256,asc}

# unpack binary + make executable
xz --decompress --force "${binname}.xz"
chown "$forgejouser" "$binname"
chmod +x "$binname"

# stop forgejo, create backup, replace binary, restart forgejo
echo "Flushing forgejo queues at $(date)"
forgejocmd manager flush-queues
echo "Stopping forgejo at $(date)"
$service_stop
echo "Creating backup in $forgejohome"
forgejocmd dump $backupopts
echo "Updating binary at $forgejobin"
cp -f "$forgejobin" "$forgejobin.bak" && mv -f "$binname" "$forgejobin"
$service_start
$service_status

echo "Upgrade to $forgejoversion successful!"

popd
