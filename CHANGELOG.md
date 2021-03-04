# Change Log
## [0.1.0] - 2021-03-04

### Added
### Changed
### Fixed

## [0.0.7] - 2021-03-03

### Added
### Changed
### Fixed


## [0.0.6] - 2021-03-03

### Added
- Adds  method to fetch account balance for account
### Changed
### Fixed

## [0.0.5] -

### Added
- Field "transfers" in "sub" of transaction events. This contains "reward" and "send" transfers containing amount and recipient information.
- Adds  method to fetch rewards for height
- New config variable "terra_lcd_addr"
### Changed
### Fixed
- fix terra addrresses decoding (for some subevents addresses were being decoded as cosmos) 

## [0.0.4] -

### Added
- Ability to change log level on the flight using http endpoint
- Added transaction log field to returned structure
### Changed
### Fixed

## [0.0.3] - 2020-11-17

### Added
- Plugin from populator that adds ability for parse the fee from raw transaction
### Changed
### Fixed

## [0.0.2] - 2020-11-02

### Added
### Changed
### Fixed
- Decoder issue after error in the beginning of the transaction list.

## [0.0.1] - 2020-10-29

Initial release

### Added
### Changed
### Fixed
