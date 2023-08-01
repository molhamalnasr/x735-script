# Overview
The "x735-script" is a Bash script designed to manage the x735 Power Management and Cooling Expansion Board (HAT) for Raspberry Pi 3 and 4. This script enables the user to control various features of the x735 board, including restarting or shutting down the Raspberry Pi using a physical button on the x735 board. Additionally, it uses the PWM pin to control the speed of the fan based on the CPU temperature, providing efficient cooling.

## Features
Control over GPIO pins to activate the restart and shutdown button on the x735 board.
Dynamic fan speed control based on the CPU temperature for optimal cooling.
Automatic installation of the x735off command, enabling safe shutdown of the Raspberry Pi with the x735 HAT.

## Installation
To install the "x735-script," follow these steps:

- Download the latest version of the package
    ``` bash
    curl -LO https://github.com/molhamalnasr/x735-script/releases/download/3.0.0/x735-script-3.0.0.deb
    ```

- Install the package:
    ``` bash
    sudo dpkg -i x735-script-3.0.0.deb
    ```

- Reboot the device:
    ``` bash
    sudo reboot
    ```

## Developer's Guide
If you are a developer and want to contribute to the "x735-script" or perform further development, follow these steps:

- Install the necessary dependencies:
    ``` bash
    sudo apt-get update && sudo apt-get install -y dpkg
    ```

- Clone the repository and change to the project directory:
    ``` bash
    git clone https://github.com/molhamalnasr/x735-script && cd x735-script
    ```

- Grant necessary permissions:
    ``` bash
    chmod +x x735-script-pkg/usr/bin/x735off
    chmod +x x735-script-pkg/usr/lib/x735-script/scripts/*.sh
    find x735-script-pkg/DEBIAN -type f ! -name 'compat' -exec chmod +x {} \;
    ```

- Build and test the package:
    ``` bash
    dpkg-deb --build x735-script-pkg
    ```

You can verify the content and installation information of the "x735-script" package with the following command:

``` bash
sudo dpkg -I x735-script-pkg.deb
```

## Release Information for Developers
For developers, please note that a new release will be automatically created when you create a new tag starting with the letter "v," for example: v3.0.1. The Github Actions file will be triggered, and it will build a new release and upload it as an artifact to Github. The release will be available under: [https://github.com/molhamalnasr/x735-script/releases/latest](https://github.com/molhamalnasr/x735-script/releases/latest).

## User Guide
For detailed instructions on how to use the "x735-script" and leverage its features, please refer to the official User Guide available at:

[https://wiki.geekworm.com/X735-script](https://wiki.geekworm.com/X735-script)

#### Contact
For any questions, issues, or support related to the "x735-script," you can reach out to the development team at:

Email: [support@geekworm.com](mailto:support@geekworm.com)

Thank you for using the "x735-script" to manage your x735 Power Management and Cooling Expansion Board. We hope this script enhances your Raspberry Pi experience and ensures smooth operations with improved cooling and power management capabilities. Should you need any assistance, feel free to contact our support team via email. Happy tinkering!