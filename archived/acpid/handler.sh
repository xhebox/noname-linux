#!/bin/sh
# Default acpi script that takes an entry for all actions

case "$1" in
	button/power)
		case "$2" in
			PBTN|PWRF) echo 'PowerButton pressed' ;;
			*) echo "ACPI action undefined: $1/$2/$3/$4" ;;
		esac
		;;
	button/sleep)
		case "$2" in
			PNP0C0E:00) echo 'Sleep resumed'
				[ -e "/bin/rfkill" ] && rfkill unblock bluetooth wifi
				;;
			SLPB|SBTN) echo 'SleepButton pressed'
				[ -e "/bin/rfkill" ] && rfkill block bluetooth wifi
				;;
			*) echo "ACPI action undefined: $1/$2/$3/$4" ;;
		esac
		;;
	ac_adapter)
		case "$2" in
			AC|ACAD|ADP0|ACPI0003:00)
				case "$4" in
					00000000) echo 'AC unpluged'
						[ -e "/bin/xbacklight" ] && DISPLAY=:0 xbacklight -set 40
						;;
					00000001) echo 'AC pluged'
						[ -e "/bin/xbacklight" ] && DISPLAY=:0 xbacklight -set 50
						;;
				esac
				;;
			*) echo "ACPI action undefined: $1/$2/$3/$4" ;;
		esac
		;;
	battery)
		case "$2" in
			BAT0)
				case "$4" in
					00000000) echo 'Battery online' ;;
					00000001) echo 'Battery offline' ;;
				esac
				;;
			CPU0)
				;;
			*)  echo "ACPI action undefined: $1/$2/$3/$4" ;;
		esac
		;;
	button/lid)
		case "$3" in
			close) echo 'LID closed'
				[ -e "/bin/loginctl" ] && loginctl hibernate
				;;
			open) echo 'LID opened' ;;
			*) echo "ACPI action undefined: $1/$2/$3/$" ;;
		esac
		;;
	*) echo "ACPI group/action undefined: $1/$2/$3/$4" ;;
esac
