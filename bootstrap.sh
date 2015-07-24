#!/bin/bash

export PLATFORM

user=b4b4r07
repo=gomi

has() {
    which "$1" >/dev/null 2>&1
    return $?
}

ink() {
    if [ "$#" -eq 0 -o "$#" -gt 2 ]; then
        echo "Usage: ink <color> <text>"
        echo "Colors:"
        echo "  black, white, red, green, yellow, blue, purple, cyan, gray"
        return 1
    fi

    local open="\033["
    local close="${open}0m"
    local black="0;30m"
    local red="1;31m"
    local green="1;32m"
    local yellow="1;33m"
    local blue="1;34m"
    local purple="1;35m"
    local cyan="1;36m"
    local gray="0;37m"
    local white="$close"

    local text="$1"
    local color="$close"

    if [ "$#" -eq 2 ]; then
        text="$2"
        case "$1" in
            black | red | green | yellow | blue | purple | cyan | gray | white)
                eval color="\$$1"
                ;;
        esac
    fi

    printf "${open}${color}${text}${close}"
}

logging() {
    if [ "$#" -eq 0 -o "$#" -gt 2 ]; then
        echo "Usage: ink <fmt> <msg>"
        echo "Formatting Options:"
        echo "  TITLE, ERROR, WARN, INFO, SUCCESS"
        return 1
    fi

    local color=
    local text="$2"

    case "$1" in
        TITLE)
            color=yellow
            ;;
        ERROR | WARN)
            color=red
            ;;
        INFO)
            color=green
            ;;
        SUCCESS)
            color=green
            ;;
        *)
            text="$1"
    esac

    timestamp() {
        ink gray "["
        ink purple "$(date +%H:%M:%S)"
        ink gray "] "
    }

    timestamp; ink "$color" "$text"; echo
}

ok() {
    logging SUCCESS "$1"
}

die() {
    logging ERROR "$1" 1>&2
}

lower() {
    if [ $# -eq 0 ]; then
        cat <&0
    elif [ $# -eq 1 ]; then
        if [ -f "$1" -a -r "$1" ]; then
            cat "$1"
        else
            echo "$1"
        fi
    else
        return 1
    fi | tr "[:upper:]" "[:lower:]"
}

ostype() {
    uname | lower
}

# os_detect export the PLATFORM variable as you see fit
os_detect() {
    export PLATFORM
    case "$(ostype)" in
        *'linux'*)  PLATFORM='linux'   ;;
        *'darwin'*) PLATFORM='darwin'  ;;
        *'bsd'*)    PLATFORM='bsd'     ;;
        *)          PLATFORM='unknown' ;;
    esac
}

main() {
    logging TITLE "== Bootstraping enhancd =="
    logging INFO "Installing dependencies..."
    sleep 1
    echo

    os_detect

    local releases i path bin re ok

    # Same as
    # curl --fail -X GET https://api.github.com/repos/b4b4r07/gomi/releases/latest | jq '.assets[0].browser_download_url' | xargs curl -L -O
    # http://stackoverflow.com/questions/24987542/is-there-a-link-to-github-for-downloading-a-file-in-the-latest-release-of-a-repo
    # http://stackoverflow.com/questions/18384873/how-to-list-the-releases-of-a-repository
    # http://stackoverflow.com/questions/5207269/releasing-a-build-artifact-on-github
    releases="$( curl -s -L https://github.com/"${user}"/"${repo}"/releases/latest |
    egrep -o '/'"${user}"'/'"${repo}"'/releases/download/[^"]*' |
    grep $PLATFORM )"

    # github releases not available
    if [ -z "$releases" ]; then
        die "URL that can be used as Github releases was not found"
        exit 1
    fi

    # download github releases for user's platform
    echo "$releases" | wget --base=http://github.com/ -i -

    # Main processing
    #
    # check machine architecture
    ok=0
    re=$(uname -m | grep -o "..$")
    for i in $releases
    do
        bin="$(basename "$i" | grep "$re")"
        if [ -n "$bin" -a -f "$bin" ]; then
            # Make a copy of repo and rename to repo
            cp "$bin" "$repo"
            chmod 755 "$repo"

            # Find the directory that you can install from $PATH
            for path in ${PATH//:/ }
            do
                logging INFO "installing to $path..."
                sudo install -m 0755 "$repo" "$path"
                if [ $? -eq 0 ]; then
                    ok "installed $repo to $path"
                    ok=1
                    break
                fi
            done

            # One binary is enough to complete this installation
            break
        fi
    done

    # no binary can execute
    if [ $ok -eq 0 ]; then
        die "there is no binary that can execute on this platform"
        echo "$releases"
        echo "go to https://github.com/$user/$repo and check how to install" 1>&2
        exit 1
    fi

    # Cleanup!
    # remove the intermediate files
    # thus complete the installation
    for i in $releases
    do
        rm -f "$(basename "$i")"
    done

    # Notification log
    if has "$repo"; then
        ok "$repo: sucessfully installed"
        # cleanup
        rm -f "$repo"
    else
        die "$repo: incomplete or unsuccessful installations"
        echo "please put ./$repo to somewhere you want" 1>&2
        echo "(on UNIX-ly systems, /usr/local/bin or the like)" 1>&2
        echo "you should run 'mv ./$repo /usr/local/bin' now" 1>&2
        exit 1
    fi
}

if echo "$-" | grep -q "i"; then
    # -> source a.sh
    return

else
    # three patterns
    # -> cat a.sh | bash
    # -> bash -c "$(cat a.sh)"
    # -> bash a.sh
    if [ -z "$BASH_VERSION" ]; then
        die "This installation requires bash"
        exit 1
    fi

    if [ "$0" = "${BASH_SOURCE:-}" ]; then
        # -> bash a.sh
        exit
    fi

    if [ -n "${BASH_EXECUTION_STRING:-}" ] || [ -p /dev/stdin ]; then
        trap "die 'terminated $0:$LINENO'; exit 1" INT ERR
        # -> cat a.sh | bash
        # -> bash -c "$(cat a.sh)"
        main
    fi
fi
