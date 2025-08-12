### cds-nutz firmware

To build the firmware you need

 - `arm-none-eabi-gcc`
 - `make`

Go to `atmel_start/gcc` and run `make`.


### Hookup

##### SPI
 - Pin A3 on the arduino is COPI (PA04, Sercom0 pad 0)
 - Pin A4 on the arduino is CS (PA05, Sercom0 pad 1)
 - Pin D8 on the arduino is CIPO (PA06, sercom0 pad 2)
 - Pin D9 on the arduino is SCK (PA07, sercom0 pad 3)