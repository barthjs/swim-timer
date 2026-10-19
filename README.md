# Swim-Timer

Swim-Timer shows the live results of an Omega timing system from Swiss Timing as overlays in streaming software
such as OBS or vMix.

## Quick start

On Linux:

```sh
curl -fsSLO https://raw.githubusercontent.com/barthjs/swim-timer/main/scripts/install.sh
bash install.sh --user
```

On Windows, in PowerShell 7:

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/barthjs/swim-timer/main/scripts/install.ps1 -OutFile install.ps1
pwsh -File install.ps1 -Mode User
```

Then open `http://localhost:8080`. A container image and Debian packages are available as well.

## Documentation

The [documentation](https://barthjs.github.io/swim-timer/) describes the installation, the use, and the development
of Swim-Timer. Every installation also serves the documentation of its own version under `/docs/`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Swim-Timer is licensed under the [MIT License](LICENSE).
