{ pkgs ?
  import (fetchTarball "http://nixos.org/channels/nixos-20.09/nixexprs.tar.xz")
  { } }:

pkgs.buildGoModule rec {
  pname = "workflow-connector";
  version = "0.2.1";
  src = ./.;
  vendorSha256 = null;
  subPackages = [ "." ];

  meta = with pkgs.lib; {
    description = "Signavio Workflow Accelerator Connector";
    homepage = "https://github.com/signavio/workflow-connector";
    license = licenses.gpl3Only;
    maintainers = with maintainers; [ sdaros ];
    platforms = platforms.linux ++ platforms.windows;
  };
}
