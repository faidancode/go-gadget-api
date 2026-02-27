import { Injectable, InternalServerErrorException } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { Resend } from 'resend';
import mjml2html from 'mjml';
import * as fs from 'fs';
import * as path from 'path';
import Handlebars from 'handlebars';

@Injectable()
export class EmailService {
  private readonly resend: Resend;
  private readonly fromEmail = 'noreply@faidancode.web.id';

  constructor(private readonly config: ConfigService) {
    const apiKey = this.config.get<string>('RESEND_API_KEY');
    if (!apiKey) {
      throw new InternalServerErrorException('RESEND_API_KEY not configured');
    }

    this.resend = new Resend(apiKey);
  }

  /* -------------------------------------------------------------------------- */
  /*                                PUBLIC API                                  */
  /* -------------------------------------------------------------------------- */

  async sendResetPasswordEmail(
    to: string,
    userName: string,
    resetLink: string,
  ) {
    const html = this.renderTemplate('reset-password', {
      userName,
      resetLink,
    });

    return this.sendEmail(to, 'Reset Your Password', html);
  }

  async sendConfirmationLink(
    to: string,
    userName: string,
    confirmLink: string,
  ) {
    const html = this.renderTemplate('confirmation', {
      userName,
      confirmLink,
      type: 'Web',
    });

    return this.sendEmail(to, 'Confirm Your Account', html);
  }

  async sendConfirmationPin(
    to: string,
    userName: string,
    confirmPin: string,
  ) {
    const html = this.renderTemplate('confirmation', {
      userName,
      confirmPin,
      type: 'Mobile',
    });

    return this.sendEmail(to, 'Confirm Your Account', html);
  }

  /* -------------------------------------------------------------------------- */
  /*                               PRIVATE CORE                                  */
  /* -------------------------------------------------------------------------- */

  private renderTemplate(
    templateName: string,
    data: Record<string, any>,
  ): string {
    const templatePath = path.join(
      process.cwd(),
      'src',
      'email',
      'templates',
      `${templateName}.mjml`,
    );

    const mjml = fs.readFileSync(templatePath, 'utf8');

    const compiled = Handlebars.compile(mjml);
    const filled = compiled(data);

    const { html, errors } = mjml2html(filled);

    if (errors?.length) {
      console.error(errors);
      throw new InternalServerErrorException('Failed to render email template');
    }

    return html;
  }

  private async sendEmail(
    to: string,
    subject: string,
    html: string,
  ): Promise<boolean> {
    try {
      const { error } = await this.resend.emails.send({
        from: `Faibook <${this.fromEmail}>`,
        to: [to],
        subject,
        html,
      });

      if (error) {
        throw new InternalServerErrorException(error.message);
      }

      return true;
    } catch (err) {
      console.error('Email send failed:', err);
      throw new InternalServerErrorException('Failed to send email');
    }
  }
}
