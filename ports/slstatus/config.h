/* See LICENSE file for copyright and license details. */

/* how often to update the statusbar (min value == 1) */
const unsigned int interval = 1000;

/* text to show if no value can be retrieved */
static const char unknown_str[] = "n/a";

/* maximum output string length */
#define MAXLEN 2048

static const struct arg args[] = {
	/* function	format        argument */
	{ wifi_icon,     "\x03%s\x01 ",            "wlp3s0" },
	{ wifi_essid,    "%s ",                    "wlp3s0" },
	{ netspeed_rx,   "%s ",                    "wlp3s0" },
	{ netspeed_tx,   "%s ",                    "wlp3s0" },
	{ cpu_perc,      "\x03\u00b3\x01 %s ",     NULL },
	{ cpu_freq,      "%s ",                    NULL },
	{ ram_perc,      "\x03\u00c6\x01 %s ",     NULL },
	{ vol_icon,   	 "\x03%s\x01 ",            NULL },
	{ vol_alsa,   	 "%s ",                    NULL },
	{ bat_icon,      "\x03%s\x01 ",            "BAT0" },
	{ battery_perc,  "%s ",                    "BAT0" },
	{ datetime,      "\x03\u00c9\x01 %s ",     "%y-%m-%d %a %H:%M:%S" },
};
