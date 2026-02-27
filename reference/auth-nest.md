
  async getCustomerByEmail(email: string) {
    const user = await this.db.query.users.findFirst({
      where: eq(schema.users.email, email),
    });

    if (!user) {
      throw new UnauthorizedException('User not found');
    }

    return {
      userId: user.id,
      role: user.role as Role,
      user: {
        name: user.name,
        email: user.email,
        ...(user.phone ? { phone: user.phone } : {}),
      },
    };
  }

  async requestPasswordReset(email: string) {
    const user = await this.db.query.users.findFirst({
      where: eq(schema.users.email, email),
      columns: { id: true, name: true },
    });

    // Security Best Practice: Selalu kembalikan respons OK/Sukses,
    // bahkan jika email tidak ditemukan, untuk mencegah enumerasi user.
    if (!user) {
      console.log(
        `[PASSWORD_RESET] Email not found: ${email}. Skipping token generation.`,
      );
      return { success: true, emailSent: false };
    }

    // Cek apakah sudah ada token aktif (belum expired) dan dibuat kurang dari 10 menit lalu
    const existingToken = await this.db.query.passwordResetTokens.findFirst({
      where: eq(schema.passwordResetTokens.userId, user.id),
      orderBy: desc(schema.passwordResetTokens.createdAt), // asumsikan ada field createdAt di tabel
    });
    const now = new Date();

    if (existingToken && existingToken.createdAt) {
      const tokenCreatedAt = new Date(existingToken.createdAt);
      const diffMinutes = (now.getTime() - tokenCreatedAt.getTime()) / 60000;

      if (diffMinutes < 10 && new Date(existingToken.expiresAt) > now) {
        // Token masih aktif dan belum mencapai jeda 10 menit
        console.log(`[PASSWORD_RESET] Request too often for userId=${user.id}`);
        return {
          success: true,
          emailSent: false,
          message:
            'A password reset link was recently sent. Please check your email or try again later.',
        };
      }
    }

    const resetToken = randomUUID();
    const expiresAt = addMinutes(new Date(), 30); // Token kedaluwarsa dalam 30 menit
    const newId = randomUUID();
    await this.db
      .insert(schema.passwordResetTokens)
      .values({
        id: newId,
        userId: user.id,
        token: resetToken,
        createdAt: now,
        expiresAt: expiresAt, // Simpan format ISO
      })
      .onDuplicateKeyUpdate({
        set: { token: resetToken, expiresAt: expiresAt },
      });

    const BASE_URL = this.configService.get<string>('WEBSTORE_URL');
    const resetUrl = `${BASE_URL}/reset-password?token=${resetToken}`;
    await this.emailService.sendResetPasswordEmail(
      email,
      user.name, // Asumsi Anda mengambil nama pengguna saat mencari user
      resetUrl,
    );

    return {
      success: true,
      emailSent: true,
    };
  }

  // --- NEW METHOD 2: Menggunakan token untuk reset password ---
  async resetPassword(token: string, newPassword: string) {
    // 1. Cari token reset
    const resetRecord = await this.db.query.passwordResetTokens.findFirst({
      where: eq(schema.passwordResetTokens.token, token),
    });

    if (!resetRecord) {
      throw new UnauthorizedException({
        success: false,
        code: 'RESET_TOKEN_INVALID',
        message: 'Reset password link is invalid or has expired.',
      });
    }

    // 2. Cek token expired
    const isExpired = isAfter(new Date(), new Date(resetRecord.expiresAt));

    if (isExpired) {
      // revoke token expired
      await this.db
        .delete(schema.passwordResetTokens)
        .where(eq(schema.passwordResetTokens.token, token));

      throw new UnauthorizedException({
        success: false,
        code: 'RESET_TOKEN_EXPIRED',
        message: 'Reset password link has expired. Please request a new one.',
      });
    }

    // 3. Ambil user
    const user = await this.db.query.users.findFirst({
      where: eq(schema.users.id, resetRecord.userId),
    });

    if (!user) {
      // revoke token jika user tidak ada
      await this.db
        .delete(schema.passwordResetTokens)
        .where(eq(schema.passwordResetTokens.token, token));

      throw new UnauthorizedException({
        success: false,
        code: 'USER_NOT_FOUND',
        message: 'User associated with this reset link no longer exists.',
      });
    }

    // 4. Hash password baru
    const passwordHash = await bcrypt.hash(newPassword, 10);

    // 5. Update password user
    await this.db
      .update(schema.users)
      .set({
        passwordHash,
        updatedAt: new Date(),
      })
      .where(eq(schema.users.id, user.id));

    // 6. Revoke token (one-time use)
    await this.db
      .delete(schema.passwordResetTokens)
      .where(eq(schema.passwordResetTokens.token, token));

    // 7. Success response
    return {
      success: true,
      message: 'Password has been reset successfully.',
    };
  }

  async requestEmailConfirmation(email: string, clientType: 'Web' | 'Mobile') {
    const user = await this.db.query.users.findFirst({
      where: eq(schema.users.email, email),
      columns: { id: true, name: true, emailConfirmed: true },
    });

    if (!user) {
      return { success: true, emailSent: false };
    }

    if (user.emailConfirmed) {
      return {
        success: true,
        emailSent: false,
        message: 'Email is already confirmed.',
      };
    }

    const existingToken = await this.db.query.emailConfirmationTokens.findFirst(
      {
        where: eq(schema.emailConfirmationTokens.userId, user.id),
        orderBy: desc(schema.emailConfirmationTokens.createdAt),
      },
    );

    const now = new Date();

    if (existingToken && existingToken.createdAt) {
      const diffMinutes =
        (now.getTime() - new Date(existingToken.createdAt).getTime()) / 60000;
      if (diffMinutes < 10 && new Date(existingToken.expiresAt) > now) {
        return {
          success: true,
          emailSent: false,
          message:
            'A confirmation email was recently sent. Please check your inbox or try again later.',
        };
      }
    }

    const token = randomUUID();
    const pin = Math.floor(100000 + Math.random() * 900000).toString(); // 6 digit pin
    const expiresAt = addMinutes(now, 60);

    await this.db
      .insert(schema.emailConfirmationTokens)
      .values({
        id: randomUUID(),
        userId: user.id,
        token,
        pin,
        expiresAt,
        createdAt: now,
      })
      .onDuplicateKeyUpdate({
        set: { token, pin, expiresAt, createdAt: now },
      });

    const baseUrl = this.configService.get<string>('WEBSTORE_URL');
    const confirmUrl = `${baseUrl}/verify-email?token=${token}`;

    if (clientType === 'Web') {
      await this.emailService.sendConfirmationLink(email, user.name, confirmUrl);
    }

    if (clientType === 'Mobile') {
      await this.emailService.sendConfirmationPin(email, user.name, pin);
    }

    return { success: true, emailSent: true };
  }

  async resendEmailConfirmation(email: string, clientType: 'Web' | 'Mobile') {
    const user = await this.db.query.users.findFirst({
      where: eq(schema.users.email, email),
      columns: { id: true, name: true, emailConfirmed: true },
    });

    // 🔐 Anti email enumeration
    if (!user) {
      return { success: true, emailSent: false };
    }

    if (user.emailConfirmed) {
      return {
        success: true,
        emailSent: false,
        message: 'Email is already confirmed.',
      };
    }

    const now = new Date();

    const existingToken = await this.db.query.emailConfirmationTokens.findFirst(
      {
        where: eq(schema.emailConfirmationTokens.userId, user.id),
        orderBy: desc(schema.emailConfirmationTokens.createdAt),
      },
    );

    // ⏱️ Throttle: 10 menit
    if (existingToken) {
      const diffMinutes =
        (now.getTime() - new Date(existingToken.createdAt).getTime()) / 60000;

      if (diffMinutes < 10 && new Date(existingToken.expiresAt) > now) {
        return {
          success: true,
          emailSent: false,
          message:
            'A confirmation email was recently sent. Please check your inbox or try again later.',
        };
      }
    }

    // ⛔ Invalidate all previous confirmation tokens for this user
    await this.db
      .delete(schema.emailConfirmationTokens)
      .where(eq(schema.emailConfirmationTokens.userId, user.id));

    // 🔑 Generate ulang
    const token = randomUUID();
    const pin = Math.floor(100000 + Math.random() * 900000).toString();
    const expiresAt = addMinutes(now, 60);

    await this.db
      .insert(schema.emailConfirmationTokens)
      .values({
        id: randomUUID(),
        userId: user.id,
        token,
        pin,
        expiresAt,
        createdAt: now,
      })
      .onDuplicateKeyUpdate({
        set: { token, pin, expiresAt, createdAt: now },
      });

    // 📩 Kirim sesuai client
    if (clientType === 'Web') {
      const baseUrl = this.configService.get<string>('WEBSTORE_URL');
      const confirmUrl = `${baseUrl}/verify-email?token=${token}`;

      await this.emailService.sendConfirmationLink(email, user.name, confirmUrl);
    }

    if (clientType === 'Mobile') {
      await this.emailService.sendConfirmationPin(email, user.name, pin);
    }

    return { success: true, emailSent: true };
  }

  async confirmEmailByToken(token: string) {
    const record = await this.db.query.emailConfirmationTokens.findFirst({
      where: eq(schema.emailConfirmationTokens.token, token),
    });

    if (!record) {
      throw new UnauthorizedException({
        success: false,
        code: 'CONFIRMATION_TOKEN_INVALID',
        message: 'Email confirmation link is invalid or has expired.',
      });
    }

    if (isAfter(new Date(), new Date(record.expiresAt))) {
      await this.db
        .delete(schema.emailConfirmationTokens)
        .where(eq(schema.emailConfirmationTokens.token, token));

      throw new UnauthorizedException({
        success: false,
        code: 'CONFIRMATION_TOKEN_EXPIRED',
        message:
          'Email confirmation link has expired. Please request a new one.',
      });
    }

    await this.db
      .update(schema.users)
      .set({ emailConfirmed: true, updatedAt: new Date() })
      .where(eq(schema.users.id, record.userId));

    await this.db
      .delete(schema.emailConfirmationTokens)
      .where(eq(schema.emailConfirmationTokens.token, token));

    return { success: true, message: 'Email has been successfully confirmed.' };
  }

  async confirmEmailByPin(email: string, pin: string) {
    const user = await this.db.query.users.findFirst({
      where: eq(schema.users.email, email),
      columns: { id: true, emailConfirmed: true },
    });

    if (!user) {
      throw new UnauthorizedException({
        success: false,
        code: 'USER_NOT_FOUND',
        message: 'User not found.',
      });
    }

    if (user.emailConfirmed) {
      return {
        success: true,
        message: 'Email is already confirmed.',
      };
    }

    const record = await this.db.query.emailConfirmationTokens.findFirst({
      where: eq(schema.emailConfirmationTokens.userId, user.id),
      orderBy: desc(schema.emailConfirmationTokens.createdAt),
    });

    if (!record || record.pin !== pin) {
      throw new UnauthorizedException({
        success: false,
        code: 'PIN_INVALID',
        message: 'Invalid confirmation PIN.',
      });
    }

    if (isAfter(new Date(), new Date(record.expiresAt))) {
      await this.db
        .delete(schema.emailConfirmationTokens)
        .where(eq(schema.emailConfirmationTokens.pin, pin));

      throw new UnauthorizedException({
        success: false,
        code: 'PIN_EXPIRED',
        message: 'Confirmation PIN has expired. Please request a new one.',
      });
    }

    await this.db
      .update(schema.users)
      .set({ emailConfirmed: true, updatedAt: new Date() })
      .where(eq(schema.users.id, user.id));

    await this.db
      .delete(schema.emailConfirmationTokens)
      .where(eq(schema.emailConfirmationTokens.pin, pin));

    return { success: true, message: 'Email has been successfully confirmed.' };
  }