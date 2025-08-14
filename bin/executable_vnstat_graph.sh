#!/bin/bash

# vnstat Daily Network Usage Graph Script
# Shows a text-based graph of total network usage (rx + tx) by day
# Requires: vnstat, jq

set -euo pipefail

# Configuration
INTERFACE=""
DAYS=30
GRAPH_WIDTH=60
SHOW_VALUES=true
UNIT="auto"
USE_COLORS=true

# Color codes - check if terminal supports colors and colors are enabled
if [ "$USE_COLORS" = true ] && [[ -t 1 ]] && { command -v tput >/dev/null 2>&1 && [ "$(tput colors 2>/dev/null || echo 0)" -ge 8 ]; }; then
    RED=$(tput setaf 1 2>/dev/null || echo '\033[0;31m')
    GREEN=$(tput setaf 2 2>/dev/null || echo '\033[0;32m') 
    YELLOW=$(tput setaf 3 2>/dev/null || echo '\033[1;33m')
    BLUE=$(tput setaf 4 2>/dev/null || echo '\033[0;34m')
    CYAN=$(tput setaf 6 2>/dev/null || echo '\033[0;36m')
    NC=$(tput sgr0 2>/dev/null || echo '\033[0m')
else
    # No colors
    RED=""
    GREEN=""
    YELLOW=""
    BLUE=""
    CYAN=""
    NC=""
fi

# Function to display usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

OPTIONS:
    -i, --interface IFACE   Network interface to monitor (default: auto-detect)
    -d, --days DAYS        Number of days to show (default: 30, max: 365)
    -w, --width WIDTH      Graph width in characters (default: 60)
    -n, --no-values        Don't show numerical values next to bars
    -c, --no-colors        Disable colored output
    -u, --unit UNIT        Unit for display: B, KB, MB, GB, auto (default: auto)
    -h, --help             Show this help message

EXAMPLES:
    $0                     # Show last 30 days for default interface
    $0 -i eth0 -d 7        # Show last 7 days for eth0
    $0 -w 40 -u GB         # 40-char wide graph in GB units
    $0 --no-values --no-colors  # Show graph without values or colors

DEPENDENCIES:
    - vnstat (for network statistics)
    - jq (for JSON parsing)
EOF
    exit 0
}

# Function to check dependencies
check_dependencies() {
    local missing_deps=()
    
    if ! command -v vnstat &> /dev/null; then
        missing_deps+=("vnstat")
    fi
    
    if ! command -v jq &> /dev/null; then
        missing_deps+=("jq")
    fi
    
    if [ ${#missing_deps[@]} -gt 0 ]; then
        echo -e "${RED}Error: Missing required dependencies:${NC}" >&2
        for dep in "${missing_deps[@]}"; do
            echo "  - $dep" >&2
        done
        echo -e "\n${YELLOW}Installation commands:${NC}" >&2
        echo "  Ubuntu/Debian: sudo apt-get install vnstat jq" >&2
        echo "  CentOS/RHEL:   sudo yum install vnstat jq" >&2
        echo "  Fedora:        sudo dnf install vnstat jq" >&2
        exit 1
    fi
}

# Function to get default interface
get_default_interface() {
    # Try to get the interface with the most traffic
    local default_iface
    local iflist_output
    
    # Get interface list and parse it properly
    iflist_output=$(vnstat --iflist 2>/dev/null || echo "")
    
    if [ -n "$iflist_output" ]; then
        # Extract interface names from the output (skip the header line)
        default_iface=$(echo "$iflist_output" | grep -v "Available interfaces:" | head -1 | awk '{print $1}')
    fi
    
    if [ -z "$default_iface" ]; then
        # Fallback: try common interface names
        for iface in wlp0s20f3 eth0 enp0s3 wlan0 wlp2s0 ens33 docker0; do
            if vnstat -i "$iface" --query &>/dev/null; then
                default_iface="$iface"
                break
            fi
        done
    fi
    
    if [ -z "$default_iface" ]; then
        echo -e "${RED}Error: No network interface found with vnstat data.${NC}" >&2
        echo -e "${YELLOW}Initialize vnstat for an interface first:${NC}" >&2
        echo "  vnstat -i <interface> --add" >&2
        echo "  systemctl start vnstat" >&2
        exit 1
    fi
    
    echo "$default_iface"
}

# Function to convert bytes to human readable format
bytes_to_human() {
    local bytes=$1
    local unit=$2
    
    case $unit in
        "B") echo "$bytes B" ;;
        "KB") echo "$(awk "BEGIN {printf \"%.1f\", $bytes/1024}") KB" ;;
        "MB") echo "$(awk "BEGIN {printf \"%.1f\", $bytes/1048576}") MB" ;;
        "GB") echo "$(awk "BEGIN {printf \"%.2f\", $bytes/1073741824}") GB" ;;
        "auto")
            if [ "$bytes" -lt 1024 ]; then
                echo "${bytes} B"
            elif [ "$bytes" -lt 1048576 ]; then
                echo "$(awk "BEGIN {printf \"%.1f\", $bytes/1024}") KB"
            elif [ "$bytes" -lt 1073741824 ]; then
                echo "$(awk "BEGIN {printf \"%.1f\", $bytes/1048576}") MB"
            else
                echo "$(awk "BEGIN {printf \"%.2f\", $bytes/1073741824}") GB"
            fi
            ;;
    esac
}

# Function to get optimal unit for dataset
get_optimal_unit() {
    local max_value=$1
    
    if [ "$max_value" -lt 1048576 ]; then
        echo "KB"
    elif [ "$max_value" -lt 1073741824 ]; then
        echo "MB"
    else
        echo "GB"
    fi
}

# Function to create the graph
create_graph() {
    local interface=$1
    local days=$2
    
    echo -e "${CYAN}Network Usage Graph for ${YELLOW}$interface${CYAN} (Last $days days)${NC}"
    echo -e "${CYAN}$(date)${NC}"
    echo ""
    
    # Get JSON data from vnstat
    local json_data
    if ! json_data=$(vnstat -i "$interface" --json d 2>/dev/null); then
        echo -e "${RED}Error: Failed to get data for interface '$interface'${NC}" >&2
        echo -e "${YELLOW}Try: vnstat -i $interface --add${NC}" >&2
        exit 1
    fi
    
    # Extract daily data and calculate totals
    local data_array=()
    local max_total=0
    
    # Parse JSON and extract last N days
    while IFS='|' read -r year month day rx tx total; do
        [ -z "$year" ] && continue
        # Skip invalid entries or headers
        [[ "$year" =~ ^[0-9]+$ ]] || continue
        [[ "$month" =~ ^[0-9]+$ ]] || continue
        [[ "$day" =~ ^[0-9]+$ ]] || continue
        
        # Format date as YYYY-MM-DD
        local formatted_date
        formatted_date=$(printf "%04d-%02d-%02d" "$year" "$month" "$day")
        data_array+=("$formatted_date|$rx|$tx|$total")
        if [ "$total" -gt "$max_total" ]; then
            max_total=$total
        fi
    done < <(echo "$json_data" | jq -r '.interfaces[0].traffic.day[]? | "\(.date.year)|\(.date.month)|\(.date.day)|\(.rx)|\(.tx)|\(.rx + .tx)"' | tail -n "$days")
    
    if [ ${#data_array[@]} -eq 0 ]; then
        echo -e "${YELLOW}No daily data available for interface $interface${NC}"
        echo -e "${YELLOW}Make sure vnstat daemon is running and has collected data${NC}"
        exit 1
    fi
    
    # Determine unit if auto
    local display_unit=$UNIT
    if [ "$display_unit" = "auto" ]; then
        display_unit=$(get_optimal_unit "$max_total")
    fi
    
    # Calculate scale factor
    local scale_factor
    scale_factor=$(awk "BEGIN {printf \"%.10f\", $GRAPH_WIDTH / $max_total}")
    
    # Print header
    printf "%-12s" "Date"
    if [ "$SHOW_VALUES" = true ]; then
        printf " %15s" "Total"
    fi
    printf " %s" "Usage"
    echo ""
    printf "%-12s" "----"
    if [ "$SHOW_VALUES" = true ]; then
        printf " %15s" "-----"
    fi
    printf " %s" "-----"
    echo ""
    
    # Print graph
    for entry in "${data_array[@]}"; do
        IFS='|' read -r date rx tx total <<< "$entry"
        
        # Format date (convert from YYYY-MM-DD to MM-DD)
        local short_date
        short_date=$(echo "$date" | cut -d'-' -f2-3)
        
        # Calculate bar length
        local bar_length
        bar_length=$(awk "BEGIN {printf \"%.0f\", $total * $scale_factor}")
        [ "$bar_length" -lt 1 ] && [ "$total" -gt 0 ] && bar_length=1
        
        # Create bar
        local bar=""
        for ((i=0; i<bar_length; i++)); do
            bar+="█"
        done
        
        # Color the bar based on usage level
        local colored_bar
        local usage_percent
        usage_percent=$(awk "BEGIN {printf \"%.0f\", ($total * 100) / $max_total}")
        
        if [ "$usage_percent" -ge 80 ]; then
            colored_bar="${RED}${bar}${NC}"
        elif [ "$usage_percent" -ge 60 ]; then
            colored_bar="${YELLOW}${bar}${NC}"
        elif [ "$usage_percent" -ge 30 ]; then
            colored_bar="${GREEN}${bar}${NC}"
        else
            colored_bar="${BLUE}${bar}${NC}"
        fi
        
        # Pad the bar to consistent width for alignment
        local padded_bar
        local padding_needed=$((GRAPH_WIDTH - bar_length))
        local padding=""
        for ((i=0; i<padding_needed; i++)); do
            padding+=" "
        done
        padded_bar="${colored_bar}${padding}"
        
        # Print the line with proper alignment
        printf "%-12s" "$short_date"
        
        if [ "$SHOW_VALUES" = true ]; then
            local human_total
            human_total=$(bytes_to_human "$total" "$display_unit")
            printf " %15s" "$human_total"
        fi
        
        printf " %s" "$padded_bar"
        printf "\n"
    done
    
    # Print footer with statistics
    echo ""
    local total_sum=0
    local day_count=${#data_array[@]}
    
    for entry in "${data_array[@]}"; do
        IFS='|' read -r date rx tx total <<< "$entry"
        total_sum=$((total_sum + total))
    done
    
    local avg_daily
    avg_daily=$((total_sum / day_count))
    
    echo -e "${CYAN}Statistics:${NC}"
    echo -e "  Period: $day_count days"
    echo -e "  Total:  $(bytes_to_human "$total_sum" "$display_unit")"
    echo -e "  Average: $(bytes_to_human "$avg_daily" "$display_unit") per day"
    echo -e "  Peak:   $(bytes_to_human "$max_total" "$display_unit")"
    
    # Legend
    echo ""
    echo -e "${CYAN}Legend:${NC}"
    echo -e "  ${BLUE}█${NC} Low usage (0-30%)   ${GREEN}█${NC} Medium usage (30-60%)"
    echo -e "  ${YELLOW}█${NC} High usage (60-80%) ${RED}█${NC} Peak usage (80-100%)"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -i|--interface)
            INTERFACE="$2"
            shift 2
            ;;
        -d|--days)
            DAYS="$2"
            if ! [[ "$DAYS" =~ ^[0-9]+$ ]] || [ "$DAYS" -lt 1 ] || [ "$DAYS" -gt 365 ]; then
                echo -e "${RED}Error: Days must be a number between 1 and 365${NC}" >&2
                exit 1
            fi
            shift 2
            ;;
        -w|--width)
            GRAPH_WIDTH="$2"
            if ! [[ "$GRAPH_WIDTH" =~ ^[0-9]+$ ]] || [ "$GRAPH_WIDTH" -lt 10 ] || [ "$GRAPH_WIDTH" -gt 200 ]; then
                echo -e "${RED}Error: Width must be a number between 10 and 200${NC}" >&2
                exit 1
            fi
            shift 2
            ;;
        -n|--no-values)
            SHOW_VALUES=false
            shift
            ;;
        -c|--no-colors)
            USE_COLORS=false
            shift
            ;;
        -u|--unit)
            UNIT="$2"
            if [[ ! "$UNIT" =~ ^(B|KB|MB|GB|auto)$ ]]; then
                echo -e "${RED}Error: Unit must be one of: B, KB, MB, GB, auto${NC}" >&2
                exit 1
            fi
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo -e "${RED}Error: Unknown option $1${NC}" >&2
            echo "Use -h or --help for usage information" >&2
            exit 1
            ;;
    esac
done

# Main execution
main() {
    check_dependencies
    
    # Get interface if not specified
    if [ -z "$INTERFACE" ]; then
        INTERFACE=$(get_default_interface)
    fi
    
    # Verify interface exists and has data
    if ! vnstat -i "$INTERFACE" --query &>/dev/null; then
        echo -e "${RED}Error: Interface '$INTERFACE' not found or has no data${NC}" >&2
        echo -e "${YELLOW}Available interfaces:${NC}" >&2
        vnstat --iflist 2>/dev/null | grep -v "Available interfaces:" || echo "  No interfaces found" >&2
        exit 1
    fi
    
    create_graph "$INTERFACE" "$DAYS"
}

# Run main function
main
