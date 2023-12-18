{
  description = "cueform";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    gomod2nix = {
      url = "github:nix-community/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.follows = "flake-utils"; 
    };
  };
  outputs = { self, nixpkgs, flake-utils, gomod2nix }:
    (flake-utils.lib.eachDefaultSystem
      (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          buildGoApplication = gomod2nix.legacyPackages.${system}.buildGoApplication;       
        in
        {
          packages.default = buildGoApplication {
            pname = "cueform";
            version = "0.1";
            src = ./.;
            pwd = ./.;
            subPackages = [ "cmd/cueform" ];
            modules = ./gomod2nix.toml;
          };
          devShells.default = pkgs.mkShell {
            buildInputs = with pkgs; [
              go
              gopls
              gotools
              gomod2nix.packages.${system}.default
           ];
          };
        })
    );
}
