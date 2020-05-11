import "./Footer.scss";

import React from "react";
import { KraftAppNavigation } from "@fider/components";
import { Fider } from "@fider/services";

export const Footer = () => {
  return (
    <div id="c-footer">
      <div className="container">
        {Fider.settings.hasLegal && (
          <div className="l-links">
            <KraftAppNavigation />
          </div>
        )}
        <a className="l-powered" target="_blank" href="https://getfider.com/">
          <img src="https://getfider.com/images/logo-100x100.png" alt="Fider" />
          <span>Powered by Fider</span>
        </a>
      </div>
    </div>
  );
};
