/* See LICENSE file for copyright and license details. */

/* how often to update the statusbar (min value == 1) */
const unsigned int interval = 1000;

/* text to show if no value can be retrieved */
static const char unknown_str[] = "n/a";

/* maximum output string length */
#define MAXLEN 2048

static const struct arg args[] = {
	/* function	format        argument */
	{ wifi_essid,    "%s ",                    "wlp3s0" },
	{ netspeed_rx,   "| %s ",                  "wlp3s0" },
	{ netspeed_tx,   " %s ",                    "wlp3s0" },
	{ cpu_perc,      "| %s ",									  NULL },
	{ cpu_freq,      " %s ",                   NULL },
	{ ram_perc,      "| %s ",										NULL },
	{ run_command, 	 "| %s ",										"amixer sget Master | grep 'Left:' | awk -F'[][]' '{ print $2 }'" },
	{ run_command, 	 " %s ",										  "amixer sget Master | grep 'Right:' | awk -F'[][]' '{ print $2 }'" },
	{ battery_perc,  "| %s ",                   "BAT0" },
	{ datetime,      "| %s ",                   "%y-%m-%d %a %H:%M:%S" },
};
