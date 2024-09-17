Name:           alpine-client
Version:        1.0.0
Release:        1%{?dist}
Summary:        Alpine Client for Linux
License:        MPL-2.0
URL:            https://alpineclient.com/
Source0:        pinnacle-linux-amd64
Source1:        %{name}.desktop
Source2:        %{name}.png
Source3:        LICENSE
Requires:       tar xrandr xdg-desktop-portal zenity

%description
Alpine Client is an all-in-one modpack for Minecraft that offers a
multitude of enhancements and optimizations to improve your gameplay.
It brings together popular mods, exclusive features, player cosmetics,
and multi-version support to curate the ultimate player experience.

%install
install -D -m 755 %{SOURCE0} %{buildroot}%{_bindir}/%{name}
install -D -m 644 %{SOURCE1} %{buildroot}%{_datadir}/applications/%{name}.desktop
install -D -m 644 %{SOURCE2} %{buildroot}%{_datadir}/icons/hicolor/256x256/apps/%{name}.png
install -D -m 644 %{SOURCE3} %{buildroot}%{_datadir}/doc/%{name}/LICENSE

%post
update-desktop-database %{_datadir}/applications

%postun
update-desktop-database %{_datadir}/applications

%files
%{_bindir}/%{name}
%{_datadir}/applications/%{name}.desktop
%{_datadir}/icons/hicolor/256x256/apps/%{name}.png
%doc %{_datadir}/doc/%{name}/LICENSE