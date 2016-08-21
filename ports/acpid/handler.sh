#!/bin/sh
# Default acpi script that takes an entry for all actions

case "$1" in
    button/power)
        case "$2" in
            PBTN|PWRF)
                echo 'PowerButton pressed'
                ;;
            *)
                echo "ACPI action undefined: $2"
                ;;
        esac
        ;;
    button/sleep)
        case "$2" in
            SLPB|SBTN)
                echo 'SleepButton pressed'
                ;;
            *)
                echo "ACPI action undefined: $2"
                ;;
        esac
        ;;
    ac_adapter)
        case "$2" in
            AC|ACAD|ADP0)
                case "$4" in
                    00000000)
                        echo 'AC unpluged'
                        ;;
                    00000001)
                        echo 'AC pluged'
                        ;;
                esac
                ;;
            *)
                echo "ACPI action undefined: $2"
                ;;
        esac
        ;;
    battery)
        case "$2" in
            BAT0)
                case "$4" in
                    00000000)
                        echo 'Battery online'
                        ;;
                    00000001)
                        echo 'Battery offline'
                        ;;
                esac
                ;;
            CPU0)
                ;;
            *)  echo "ACPI action undefined: $2" ;;
        esac
        ;;
    button/lid)
        case "$3" in
            close)
                echo 'LID closed'
                ;;
            open)
                echo 'LID opened'
                ;;
            *)
                echo "ACPI action undefined: $3"
                ;;
    esac
    ;;
    *)
        echo "ACPI group/action undefined: $1 / $2"
        ;;
esac
