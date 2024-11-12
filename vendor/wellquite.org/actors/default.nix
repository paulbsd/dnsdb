{ pkgs ? import <nixpkgs> {} }:

with pkgs;

buildGoModule {
  pname = "actors";
  version = "latest";

  src = with builtins; filterSource
    (path: type: substring 0 1 (baseNameOf path) != "." && (baseNameOf path) != "default.nix" && type != "symlink")
    ./.;

  vendorSha256 = "sha256-Hf0peGK3BBScmGKNFsTN6HziOdptrBdFORbDGDHDx3U=";

  meta = with lib; {
    description = "Actor library for Go";
    homepage = "https://fossil.wellquite.org/actors";
    license = licenses.asl20;
    platforms = platforms.linux ++ platforms.darwin;
  };
}
