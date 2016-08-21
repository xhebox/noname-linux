/* See LICENSE file for copyright and license details. */

/* how often to update the statusbar (min value == 1) */
#define UPDATE_INTERVAL 1

/* text to show if no value can be retrieved */
#define UNKNOWN_STR     "n/a"

static const struct arg args[] = {
	/* function	format        argument */
	{ cpu_perc,     "[ Cpu %s| ",     NULL },
	{ ram_perc,     "Ram %s| ",   NULL },
	{ battery_perc, "BAT %s| ",   "BAT0" },
	{ wifi_perc,    "Wifi %s| ",   "wlp3s0" },
	{ datetime,     "%s]", "%y-%m-%d %a %H:%M:%S" },
};
