#!/usr/bin/env bash
set -euo pipefail

APP_NAME="ledlight"
APP_USER="ledlight"
APP_GROUP="ledlight"
APP_DIR="/opt/ledlight"
ASSET_ROOT=""
LIVE_DIR=""
LOG_DIR="/var/log/ledlight"
OUTPUT="single"
IFACE=""
AUX_IFACE=""
ARCH="$(uname -m)"
ENV_SOURCE=""
DEFAULT_SLIDE_SOURCE=""
SKIP_BUILD=false
ASSUME_YES=false
START_SERVICE="ask"

REPO_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
FALLBACK_SOURCE="${REPO_ROOT}/misc/fallback.png"
SERVICE_SOURCE="${REPO_ROOT}/misc/ledlight.service"
ENV_EXAMPLE="${REPO_ROOT}/.env.example"
DIST_BINARY="${REPO_ROOT}/dist/run"

usage() {
	cat <<USAGE
Usage: ./setup.sh [options]

Initial Linux server setup for Ledlight. Run this from the repository on the target server.
The application directory is fixed at /opt/ledlight because the fallback image path is fixed in the binary.

Options:
  --asset-root PATH              Asset root directory. Default: ~/ledlight-assets
  --live-dir PATH                Live asset directory. Default: <asset-root>/live
  --env PATH                     Existing .env file to install. Default: keep existing or generate
  --deafult-slide PATH           Default slide image to install. Default: misc/fallback.png
  --output single|dual           Display output mode. Default: single
  --iface NAME                   Primary network interface. Default: autodetect
  --aux-iface NAME               Auxiliary interface for dual output. Default: autodetect
  --arch amd64|arm64             Go target architecture when building. Default: host architecture
  --skip-build                   Use existing dist/run instead of building
  --start                        Enable and start/restart the systemd service
  --no-start                     Install only; do not start the service
  -y, --yes                      Non-interactive defaults where possible
  -h, --help                     Show this help

Examples:
  sudo ./setup.sh --start
  sudo ./setup.sh --output dual --start
USAGE
}

die() {
	printf 'ERROR: %s\n' "$*" >&2
	exit 1
}

info() {
	printf '==> %s\n' "$*"
}

need_arg() {
	[ "$#" -ge 2 ] || die "$1 requires a value."
}

run_as_root() {
	if [ "$(id -u)" -eq 0 ]; then
		"$@"
	else
		sudo "$@"
	fi
}

confirm() {
	local prompt="$1"
	local default="${2:-no}"
	local reply

	if [ "$ASSUME_YES" = true ]; then
		return 0
	fi

	if [ ! -t 0 ]; then
		[ "$default" = "yes" ]
		return
	fi

	if [ "$default" = "yes" ]; then
		read -r -p "${prompt} [Y/n] " reply
		[ -z "$reply" ] || [[ "$reply" =~ ^[Yy]$ ]]
	else
		read -r -p "${prompt} [y/N] " reply
		[[ "$reply" =~ ^[Yy]$ ]]
	fi
}

prompt_value() {
	local var_name="$1"
	local prompt="$2"
	local default="${3:-}"
	local value

	if [ "$ASSUME_YES" = true ]; then
		value="$default"
	elif [ -t 0 ]; then
		if [ -n "$default" ]; then
			read -r -p "${prompt} [${default}]: " value
			value="${value:-$default}"
		else
			read -r -p "${prompt}: " value
		fi
	else
		value="$default"
	fi

	printf -v "$var_name" '%s' "$value"
}

normalize_arch() {
	case "$ARCH" in
		x86_64 | amd64) ARCH="amd64" ;;
		aarch64 | arm64) ARCH="arm64" ;;
		*) die "Unsupported architecture '${ARCH}'. Pass --arch amd64 or --arch arm64." ;;
	esac
}

package_installed() {
	local package="$1"

	if command -v apt-get >/dev/null 2>&1; then
		dpkg-query -W -f='${Status}' "$package" 2>/dev/null | grep -q 'install ok installed'
	elif command -v dnf >/dev/null 2>&1; then
		rpm -q "$package" >/dev/null 2>&1
	elif command -v pacman >/dev/null 2>&1; then
		pacman -Q "$package" >/dev/null 2>&1
	else
		return 1
	fi
}

install_missing_packages() {
	local packages=()
	local required_tools=(ca-certificates tzdata)

	if [ "$SKIP_BUILD" = false ]; then
		required_tools+=(go)
	fi

	if command -v apt-get >/dev/null 2>&1; then
		command -v ip >/dev/null 2>&1 || packages+=(iproute2)
		command -v setcap >/dev/null 2>&1 || packages+=(libcap2-bin)
		command -v getcap >/dev/null 2>&1 || packages+=(libcap2-bin)
		command -v systemctl >/dev/null 2>&1 || packages+=(systemd)

		for tool in "${required_tools[@]}"; do
			case "$tool" in
				go) package_installed golang-go || packages+=(golang-go) ;;
				*) package_installed "$tool" || packages+=("$tool") ;;
			esac
		done

		if [ "${#packages[@]}" -gt 0 ]; then
			info "Installing missing packages: ${packages[*]}"
			run_as_root apt-get update
			run_as_root apt-get install -y "${packages[@]}"
		fi
	elif command -v dnf >/dev/null 2>&1; then
		command -v ip >/dev/null 2>&1 || packages+=(iproute)
		command -v setcap >/dev/null 2>&1 || packages+=(libcap)
		command -v getcap >/dev/null 2>&1 || packages+=(libcap)
		command -v systemctl >/dev/null 2>&1 || packages+=(systemd)

		for tool in "${required_tools[@]}"; do
			package_installed "$tool" || packages+=("$tool")
		done

		if [ "${#packages[@]}" -gt 0 ]; then
			info "Installing missing packages: ${packages[*]}"
			run_as_root dnf install -y "${packages[@]}"
		fi
	elif command -v pacman >/dev/null 2>&1; then
		command -v ip >/dev/null 2>&1 || packages+=(iproute2)
		command -v setcap >/dev/null 2>&1 || packages+=(libcap)
		command -v getcap >/dev/null 2>&1 || packages+=(libcap)
		command -v systemctl >/dev/null 2>&1 || packages+=(systemd)

		for tool in "${required_tools[@]}"; do
			case "$tool" in
				ca-certificates) package_installed ca-certificates || packages+=(ca-certificates) ;;
				go) package_installed go || packages+=(go) ;;
				*) package_installed "$tool" || packages+=("$tool") ;;
			esac
		done

		if [ "${#packages[@]}" -gt 0 ]; then
			info "Installing missing packages: ${packages[*]}"
			run_as_root pacman -Sy --needed --noconfirm "${packages[@]}"
		fi
	else
		die "Unsupported package manager. Install ca-certificates, go if building on-server, iproute2, libcap, systemd, and tzdata manually."
	fi
}

ensure_linux_systemd() {
	[ "$(uname -s)" = "Linux" ] || die "Ledlight must be installed on Linux."
	install_missing_packages
	command -v systemctl >/dev/null 2>&1 || die "systemctl is required."
	command -v ip >/dev/null 2>&1 || die "ip command is required."
	command -v setcap >/dev/null 2>&1 || die "setcap is required."
	command -v getcap >/dev/null 2>&1 || die "getcap is required."
}

build_binary() {
	if [ "$SKIP_BUILD" = true ]; then
		[ -x "$DIST_BINARY" ] || die "dist/run is missing and --skip-build was provided."
		return
	fi

	command -v go >/dev/null 2>&1 || die "Go is required to build dist/run. Copy a prebuilt binary to dist/run or install Go."

	info "Building Linux ${ARCH} binary at dist/run"
	mkdir -p "${REPO_ROOT}/dist"
	(
		cd "$REPO_ROOT"
		GOOS=linux GOARCH="$ARCH" go build -o "$DIST_BINARY" ./src
	)
}

show_interfaces() {
	if command -v ip >/dev/null 2>&1; then
		ip -br link || true
	fi
}

home_dir() {
	if [ -n "${SUDO_USER:-}" ] && [ "$SUDO_USER" != "root" ]; then
		getent passwd "$SUDO_USER" | cut -d: -f6
	else
		printf '%s\n' "${HOME:-/root}"
	fi
}

expand_path() {
	case "$1" in
		"~") home_dir ;;
		"~/"*) printf '%s/%s\n' "$(home_dir)" "${1#~/}" ;;
		*) printf '%s\n' "$1" ;;
	esac
}

detect_interfaces() {
	local detected
	mapfile -t detected < <(
		ip -o link show |
			awk -F': ' '$2 != "lo" { print $2 }' |
			sed 's/@.*//' |
			grep -Ev '^(docker|br-|veth|virbr|vmnet|tun|tap|wg|tailscale|zt|lo)' || true
	)

	if [ "$OUTPUT" = "single" ]; then
		if [ -z "$IFACE" ]; then
			[ "${#detected[@]}" -ge 1 ] || die "No candidate network interface found. Pass --iface NAME."
			IFACE="${detected[0]}"
		fi
	else
		if [ -z "$IFACE" ] || [ -z "$AUX_IFACE" ]; then
			[ "${#detected[@]}" -ge 2 ] || die "Dual output needs two candidate network interfaces. Pass --iface NAME --aux-iface NAME."
			IFACE="${IFACE:-${detected[0]}}"
			AUX_IFACE="${AUX_IFACE:-${detected[1]}}"
		fi
	fi
}

collect_configuration() {
	ASSET_ROOT="$(expand_path "${ASSET_ROOT:-~/ledlight-assets}")"
	LIVE_DIR="$(expand_path "${LIVE_DIR:-${ASSET_ROOT}/live}")"
	if [ -n "$ENV_SOURCE" ]; then
		ENV_SOURCE="$(expand_path "$ENV_SOURCE")"
	fi
	if [ -n "$DEFAULT_SLIDE_SOURCE" ]; then
		DEFAULT_SLIDE_SOURCE="$(expand_path "$DEFAULT_SLIDE_SOURCE")"
	fi
	DEFAULT_SLIDE_SOURCE="${DEFAULT_SLIDE_SOURCE:-$FALLBACK_SOURCE}"

	if [ "$OUTPUT" != "single" ] && [ "$OUTPUT" != "dual" ]; then
		die "--output must be single or dual."
	fi

	detect_interfaces

	[ -n "$IFACE" ] || die "Primary interface is required. Pass --iface NAME."

	if [ "$OUTPUT" = "dual" ]; then
		[ -n "$AUX_IFACE" ] || die "Auxiliary interface is required for dual output. Pass --aux-iface NAME."
	fi

	info "Using primary interface: ${IFACE}"
	if [ "$OUTPUT" = "dual" ]; then
		info "Using auxiliary interface: ${AUX_IFACE}"
	fi

	[ -f "$FALLBACK_SOURCE" ] || die "Fallback image not found at ${FALLBACK_SOURCE}."
	[ -f "$SERVICE_SOURCE" ] || die "Service file not found at ${SERVICE_SOURCE}."
	[ -f "$DEFAULT_SLIDE_SOURCE" ] || die "Default slide source not found at ${DEFAULT_SLIDE_SOURCE}."

	if [ -n "$ENV_SOURCE" ]; then
		[ -f "$ENV_SOURCE" ] || die "Environment source not found at ${ENV_SOURCE}."
	elif [ ! -f "$ENV_EXAMPLE" ]; then
		die ".env.example not found at ${ENV_EXAMPLE}."
	fi
}

create_service_user() {
	if ! getent group "$APP_GROUP" >/dev/null 2>&1; then
		info "Creating ${APP_GROUP} service group"
		run_as_root groupadd --system "$APP_GROUP"
	fi

	if id "$APP_USER" >/dev/null 2>&1; then
		return
	fi

	local nologin_shell
	nologin_shell="$(command -v nologin || command -v false || printf '/usr/sbin/nologin')"

	info "Creating ${APP_USER} service user"
	run_as_root useradd --system --gid "$APP_GROUP" --home "$APP_DIR" --shell "$nologin_shell" "$APP_USER"
}

install_files() {
	info "Creating application, asset, and log directories"
	run_as_root mkdir -p "$APP_DIR" "$ASSET_ROOT" "$LIVE_DIR" "$LOG_DIR"

	info "Installing binary and image assets"
	run_as_root install -m 0755 "$DIST_BINARY" "${APP_DIR}/run"
	run_as_root install -m 0644 "$FALLBACK_SOURCE" "${APP_DIR}/fallback.png"
	run_as_root install -m 0644 "$DEFAULT_SLIDE_SOURCE" "${ASSET_ROOT}/default.png"

	if [ -n "$ENV_SOURCE" ]; then
		info "Installing environment file from ${ENV_SOURCE}"
		run_as_root install -m 0640 "$ENV_SOURCE" "${APP_DIR}/.env"
	elif run_as_root test -f "${APP_DIR}/.env"; then
		info "Keeping existing ${APP_DIR}/.env"
	else
		info "Generating ${APP_DIR}/.env"
		local tmp_env
		tmp_env="$(mktemp)"
		awk \
			-v output="$OUTPUT" \
			-v iface="$IFACE" \
			-v aux_iface="${AUX_IFACE:-enp2s0}" \
			-v live_dir="$LIVE_DIR" \
			-v default_slide="${ASSET_ROOT}/default.png" \
			'
			BEGIN {
				replacements["DEVICE_OUTPUT"] = "\"" output "\""
				replacements["DEVICE_IFACE"] = "\"" iface "\""
				replacements["DEVICE_AUX_IFACE"] = "\"" aux_iface "\""
				replacements["ASSET_PATH"] = "\"" live_dir "\""
				replacements["DEFAULT_SLIDE"] = "\"" default_slide "\""
			}
			{
				key = $0
				sub(/=.*/, "", key)
				if (key in replacements) {
					print key "=" replacements[key]
				} else {
					print
				}
			}
			' "$ENV_EXAMPLE" > "$tmp_env"
		run_as_root install -m 0640 "$tmp_env" "${APP_DIR}/.env"
		rm -f "$tmp_env"
	fi

	info "Installing systemd unit"
	run_as_root install -m 0644 "$SERVICE_SOURCE" "/etc/systemd/system/${APP_NAME}.service"
}

configure_permissions() {
	info "Setting ownership and raw socket capability"
	run_as_root chown -R "${APP_USER}:${APP_GROUP}" "$APP_DIR" "$LOG_DIR" "$ASSET_ROOT"
	run_as_root chmod 0755 "$ASSET_ROOT" "$LIVE_DIR"
	run_as_root chmod 0755 "${APP_DIR}/run"
	run_as_root setcap cap_net_raw+ep "${APP_DIR}/run"
	getcap "${APP_DIR}/run"

	if [ -n "${SUDO_USER:-}" ] && [ "$SUDO_USER" != "root" ]; then
		local sudo_home
		sudo_home="$(home_dir)"
		if [ -d "$sudo_home" ] && ! run_as_root namei -m "$LIVE_DIR" | awk 'NR > 1 && $1 !~ /x/ { exit 1 }'; then
			info "Warning: ${APP_USER} may not be able to traverse ${sudo_home}; move assets or relax directory execute permissions if assets cannot be read."
		fi
	fi
}

bring_interfaces_up() {
	info "Bringing configured network interfaces up"
	run_as_root ip link set "$IFACE" up

	if [ "$OUTPUT" = "dual" ]; then
		run_as_root ip link set "$AUX_IFACE" up
	fi
}

reload_and_maybe_start_service() {
	info "Reloading systemd"
	run_as_root systemctl daemon-reload

	case "$START_SERVICE" in
		yes)
			run_as_root systemctl enable "$APP_NAME"
			run_as_root systemctl restart "$APP_NAME"
			;;
		no)
			info "Service installed but not started. Start it with: sudo systemctl enable --now ${APP_NAME}"
			;;
		ask)
			if confirm "Enable and start ${APP_NAME} now?" "yes"; then
				run_as_root systemctl enable "$APP_NAME"
				run_as_root systemctl restart "$APP_NAME"
			else
				info "Service installed but not started. Start it with: sudo systemctl enable --now ${APP_NAME}"
			fi
			;;
		*) die "Invalid START_SERVICE value '${START_SERVICE}'." ;;
	esac
}

while [ "$#" -gt 0 ]; do
	case "$1" in
		--asset-root) need_arg "$@"; ASSET_ROOT="$2"; shift 2 ;;
		--live-dir) need_arg "$@"; LIVE_DIR="$2"; shift 2 ;;
		--env) need_arg "$@"; ENV_SOURCE="$2"; shift 2 ;;
		--deafult-slide) need_arg "$@"; DEFAULT_SLIDE_SOURCE="$2"; shift 2 ;;
		--default-slide) need_arg "$@"; DEFAULT_SLIDE_SOURCE="$2"; shift 2 ;;
		--output) need_arg "$@"; OUTPUT="$2"; shift 2 ;;
		--iface) need_arg "$@"; IFACE="$2"; shift 2 ;;
		--aux-iface) need_arg "$@"; AUX_IFACE="$2"; shift 2 ;;
		--arch) need_arg "$@"; ARCH="$2"; shift 2 ;;
		--skip-build) SKIP_BUILD=true; shift ;;
		--start) START_SERVICE="yes"; shift ;;
		--no-start) START_SERVICE="no"; shift ;;
		-y | --yes) ASSUME_YES=true; shift ;;
		-h | --help) usage; exit 0 ;;
		*) die "Unknown option: $1" ;;
	esac
done

normalize_arch

ensure_linux_systemd
collect_configuration
build_binary
create_service_user
install_files
configure_permissions
bring_interfaces_up
reload_and_maybe_start_service

info "Setup complete"
info "Environment: ${APP_DIR}/.env"
info "Application logs: ${LOG_DIR}/$(date +%F).log"
info "systemd logs: journalctl -u ${APP_NAME} -f"
