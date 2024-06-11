01001000	H	reserved for HELP  
01001001	I	reserved for IPX and ISIZE  
01010000	P	reserved for PX  
01010011	S	reserved for SIZE  

INFO  
0001 0000															get info about all canvasses  
SIZE canvas  
0010 xxxx																get size of canvas x, returns itself plus 4 bytes of size  
PX   canvas x      y      color  
1000 xxxx   2 byte 2 byte								for getting pixel value, returns itself plus 3 bytes containing r,g,b  
1001 xxxx   2 byte 2 byte 1 byte				for grayscale  
1010 xxxx   2 byte 2 byte 2 byte				for rgba half-precision  
1011 xxxx   2 byte 2 byte 3 byte				for rgb  
1100 xxxx   2 byte 2 byte 4 byte				for rgba  
SPX  canvas sfx    note   volume  
1110 xxxx   1 byte 2 byte 1 byte				play sound loop  
1111 xxxx   1 byte 2 byte 1 byte				play sound once  


to set the pixels (0,0), (1,0), (0,1), (1,1) to red,green,blue,white on canvas 0 you can send  
0xC0 0x00 0x00 0x00 0x00 0xff 0x00 0x00 0xff // uses the set RGBA on pixel 0,0. sets the pixel to #ff0000 with blending  
0xB0 0x00 0x01 0x00 0x00 0x00 0xff 0x00      // uses the set RGB on pixel 1,0. sets the pixel to #00ff00  
0xA0 0x00 0x00 0x00 0x01 0x00 0xff           // uses the set half RGBA on pixel 0,1. sets the pixel to #00f with blending  
0x90 0x00 0x01 0x00 0x01 0xff                // uses the set whitespace on pixel 1,1 to set the pixel to #ff  
  
without spaces, comments or newlines  
