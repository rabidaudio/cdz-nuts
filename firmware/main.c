#include <atmel_start.h>

#include "tusb.h"

#include <peripheral_clk_config.h>
#include <hal_init.h>
#include <hpl_gclk_base.h>
#include <hpl_pm_base.h>

void tud_hw_init();

void my_spi_init();

int main(void)
{
    /* Initializes MCU, drivers and middleware */
    NVMCTRL->CTRLB.bit.RWS = 1;
    atmel_start_init();

    my_spi_init();

    // Initialize usb stack
    tud_hw_init();
    tusb_init();

    /* Replace with your application code */
    while (1) {
        // handle USB stuff
        tud_task();

        // handle SPI stuff
        SERCOM0->SPI.DATA.reg = 'b';
    }
}


void my_spi_init()
{
    // setup for SPI peripheral operation
    SERCOM0->SPI.CTRLA.bit.MODE = 2;

    // Set DIPO. PAD[0] is data in
    SERCOM0->SPI.CTRLA.bit.DIPO = 0;

    // Set DOPO. PAD[1] is chip select, PAD[2] is data out, PAD[3] is sck.
    SERCOM0->SPI.CTRLA.bit.DOPO = 1;

    // Enable slave data preloading
    SERCOM0->SPI.CTRLB.bit.PLOADEN = 1;

    // enable the SPI
    SERCOM0->SPI.CTRLB.bit.RXEN = 1;
    SERCOM0->SPI.CTRLA.bit.ENABLE = 1;
}

void tud_hw_init()
{
    /* USB Clock init
     * The USB module requires a GCLK_USB of 48 MHz ~ 0.25% clock
     * for low speed and full speed operation. */
    _pm_enable_bus_clock(PM_BUS_APBB, USB);
    _pm_enable_bus_clock(PM_BUS_AHB, USB);
    _gclk_enable_channel(USB_GCLK_ID, GCLK_CLKCTRL_GEN_GCLK0_Val);

    // USB Pin Init
    gpio_set_pin_direction(PIN_PA24, GPIO_DIRECTION_OUT);
    gpio_set_pin_level(PIN_PA24, false);
    gpio_set_pin_pull_mode(PIN_PA24, GPIO_PULL_OFF);
    gpio_set_pin_direction(PIN_PA25, GPIO_DIRECTION_OUT);
    gpio_set_pin_level(PIN_PA25, false);
    gpio_set_pin_pull_mode(PIN_PA25, GPIO_PULL_OFF);

    gpio_set_pin_function(PIN_PA24, PINMUX_PA24G_USB_DM);
    gpio_set_pin_function(PIN_PA25, PINMUX_PA25G_USB_DP);
}

void USB_Handler() {
    tud_int_handler(0);
}
