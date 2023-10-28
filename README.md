# cs2-demos
Takes a folder of CS2 demos and creates a spreadsheet with stats

Credit:
Demo file parsing - https://github.com/markus-wa/demoinfocs-golang

## Spreadsheet generation times
| # of demos 	| Total file size 	| Single-core  	| Multi-core     	|
|------------	|-----------------	|--------------	|----------------	|
| 200        	| ~15gbs          	| 700+ seconds 	| 90-120 seconds 	|
| 60         	| ~5gbs           	| 500+ seconds 	| 50-55 seconds  	|
| 14         	| ~1gbs           	| 90+ seconds  	| 10-17 seconds  	|
| 5          	| ~500mbs         	| 20 seconds   	| 5 seconds      	|
