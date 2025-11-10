"""Add validated_tribunal flag to exam

Revision ID: 2f4b772a1c9d
Revises: 0d83c7bda89a
Create Date: 2025-11-09 19:20:00.000000
"""

from alembic import op
import sqlalchemy as sa


revision = "2f4b772a1c9d"
down_revision = "0d83c7bda89a"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.add_column(
        "exam",
        sa.Column(
            "validated_tribunal",
            sa.Boolean(),
            nullable=False,
            server_default=sa.sql.expression.false(),
        ),
    )
    op.alter_column("exam", "validated_tribunal", server_default=None)


def downgrade() -> None:
    op.drop_column("exam", "validated_tribunal")

