{ pkgs, ... }:

{
  cachix.pull = [ "sestrella" ];

  packages = [ pkgs.iecs ];
}
